package btcdcommander

import (
	"code.google.com/p/go.net/websocket"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conformal/btcjson"
	"github.com/conformal/btcwire"
	"github.com/conformal/btcws"
	"github.com/conformal/go-socks"
	"io/ioutil"
	"time"
)

// ErrBtcdDisconnected describes an error where an operation cannot
// successfully complete due to btcwallet not being connected to
// btcd.
var ErrBtcdDisconnected = btcjson.Error{
	Code:    -1,
	Message: "btcd disconnected",
}

type Config struct {
	CAFileName string
	Connect    string
	Username   string
	Password   string
	Proxy      string
	ProxyUser  string
	ProxyPass  string
	MainNet    bool
}

func (c *Config) Net() btcwire.BitcoinNet {
	if c.MainNet {
		return btcwire.MainNet
	}
	return btcwire.TestNet3
}

type NotificationListener interface {
	BtcdConnected()
	BtcdDisconnected()
	BlockConnected(blockHash string)
	BlockDisconnected(blockHash string)
}

type Commander struct {
	cfg           *Config
	listener      NotificationListener
	newJSONID     chan uint64
	lastJSONID    uint64
	rpc           *rpcConn
	accessRpcConn chan *AccessCurrentRpcConn
	running       bool
}

type ServerRequest struct {
	request  btcjson.Cmd
	result   interface{}
	response chan *ServerResponse
}

type ServerResponse struct {
	result interface{}
	err    *btcjson.Error
}

type addRPCRequest struct {
	Request      *ServerRequest
	ResponseChan chan chan *ServerResponse
}

type rpcConn struct {
	ws         *websocket.Conn
	addRequest chan *addRPCRequest
	closed     chan struct{}
	listener   NotificationListener
}

func (btcd *rpcConn) connected() bool {
	select {
	case <-btcd.closed:
		return false

	default:
		return true
	}
}

func (btcd *rpcConn) sendRequest(cmd btcjson.Cmd, result interface{}) chan *ServerResponse {
	request := &ServerRequest{
		request:  cmd,
		result:   result,
		response: make(chan *ServerResponse, 1),
	}

	select {
	case <-btcd.closed:
		// The connection has closed, so instead of adding and sending
		// a request, return a channel that just replies with the
		// error for a disconnected btcd.
		responseChan := make(chan *ServerResponse, 1)
		response := &ServerResponse{
			err: &ErrBtcdDisconnected,
		}
		responseChan <- response
		return responseChan

	default:
		addRequest := &addRPCRequest{
			Request:      request,
			ResponseChan: make(chan chan *ServerResponse, 1),
		}
		btcd.addRequest <- addRequest
		return <-addRequest.ResponseChan
	}
}

func (btcd *rpcConn) send(sr *ServerRequest) error {
	mrequest, err := sr.request.MarshalJSON()
	if err != nil {
		return err
	}
	return websocket.Message.Send(btcd.ws, mrequest)
}

func (btcd *rpcConn) notifyListener(cmd btcjson.Cmd) {
	switch cmd.Method() {
	case btcws.BlockConnectedNtfnMethod:
		bcn, ok := cmd.(*btcws.BlockConnectedNtfn)
		if !ok {
			return
		}
		btcd.listener.BlockConnected(bcn.Hash)
	case btcws.BlockDisconnectedNtfnMethod:
		bdn, ok := cmd.(*btcws.BlockDisconnectedNtfn)
		if !ok {
			return
		}
		btcd.listener.BlockDisconnected(bdn.Hash)
	default:
		break
	}
}

type receivedResponse struct {
	id    uint64
	raw   string
	reply *btcjson.Reply
}

func (btcd *rpcConn) start() {
	done := btcd.closed
	responses := make(chan *receivedResponse)
	notifications := make(chan btcjson.Cmd)

	// Maintain a map of JSON IDs to RPCRequests currently being waited on.
	go func() {
		m := make(map[uint64]*ServerRequest)
		for {
			select {
			case addrequest := <-btcd.addRequest:
				rpcrequest := addrequest.Request
				m[rpcrequest.request.Id().(uint64)] = rpcrequest

				if err := btcd.send(rpcrequest); err != nil {
					// Connection lost.
					btcd.ws.Close()
				}

				addrequest.ResponseChan <- rpcrequest.response

			case recvResponse := <-responses:
				rpcrequest, ok := m[recvResponse.id]
				if !ok {
					log.Warnf("Received unexpected btcd response")
					continue
				}
				delete(m, recvResponse.id)

				// If no result var was set, create and send
				// send the response unmarshaled by the json
				// package.
				if rpcrequest.result == nil {
					response := &ServerResponse{
						result: recvResponse.reply.Result,
						err:    recvResponse.reply.Error,
					}
					rpcrequest.response <- response
					continue
				}

				// A return var was set, so unmarshal again
				// into the var before sending the response.
				r := &btcjson.Reply{
					Result: rpcrequest.result,
				}
				json.Unmarshal([]byte(recvResponse.raw), &r)
				response := &ServerResponse{
					result: r.Result,
					err:    r.Error,
				}
				rpcrequest.response <- response

			case recvNotification := <-notifications:
				btcd.notifyListener(recvNotification)

			case <-done:
				for _, request := range m {
					response := &ServerResponse{
						err: &ErrBtcdDisconnected,
					}
					request.response <- response
				}
				return
			}
		}
	}()

	// Listen for replies/notifications from btcd, and decide how to handle them.
	go func() {
		for {
			var m string
			if err := websocket.Message.Receive(btcd.ws, &m); err != nil {
				log.Debugf("Cannot receive btcd message: %v", err)
				// TODO: close of closed channel
				close(done)
				return
			}

			n, err := unmarshalNotification(m)
			if err == nil {
				notifications <- n
				continue
			}

			// Must be a response.
			r, err := unmarshalResponse(m)
			if err == nil {
				responses <- r
				continue
			}

			// Not sure what was received but it isn't correct.
			log.Warnf("Received invalid message from btcd")
		}
	}()
}

// unmarshalResponse attempts to unmarshal a marshaled JSON-RPC
// response.
func unmarshalResponse(s string) (*receivedResponse, error) {
	var r btcjson.Reply
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return nil, err
	}

	// Check for a valid ID.
	if r.Id == nil {
		return nil, errors.New("id is nil")
	}
	fid, ok := (*r.Id).(float64)
	if !ok {
		return nil, errors.New("id is not a number")
	}
	response := &receivedResponse{
		id:    uint64(fid),
		raw:   s,
		reply: &r,
	}
	return response, nil
}

// unmarshalNotification attempts to unmarshal a marshaled JSON-RPC
// notification (Request with a nil or no ID).
func unmarshalNotification(s string) (btcjson.Cmd, error) {
	req, err := btcjson.ParseMarshaledCmd([]byte(s))
	if err != nil {
		return nil, err
	}

	if req.Id() != nil {
		return nil, errors.New("id is non-nil")
	}

	return req, nil
}

func newBtcdRpcConn(cfg *Config, listener NotificationListener) (*rpcConn, error) {
	ws, err := newBtcdWS(cfg)
	if err != nil {
		return nil, err
	}

	r := &rpcConn{
		ws:         ws,
		addRequest: make(chan *addRPCRequest),
		closed:     make(chan struct{}),
		listener:   listener,
	}
	r.start()
	return r, nil
}

// newBtcdWS opens a websocket connection to a btcd instance.
func newBtcdWS(cfg *Config) (*websocket.Conn, error) {
	certificates, err := ioutil.ReadFile(cfg.CAFileName)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("wss://%s/ws", cfg.Connect)
	config, err := websocket.NewConfig(url, "https://localhost/")
	if err != nil {
		return nil, err
	}

	// btcd uses a self-signed TLS certifiate which is used as the CA.
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(certificates)
	config.TlsConfig = &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}

	// btcd requires basic authorization, so set the Authorization header.
	login := cfg.Username + ":" + cfg.Password
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
	config.Header.Add("Authorization", auth)

	// Dial connection.
	var ws *websocket.Conn
	var cerr error
	if cfg.Proxy != "" {
		proxy := &socks.Proxy{
			Addr:     cfg.Proxy,
			Username: cfg.ProxyUser,
			Password: cfg.ProxyPass,
		}
		conn, err := proxy.Dial("tcp", cfg.Connect)
		if err != nil {
			return nil, err
		}

		tlsConn := tls.Client(conn, config.TlsConfig)
		ws, cerr = websocket.NewClient(config, tlsConn)
	} else {
		ws, cerr = websocket.DialConfig(config)
	}
	if cerr != nil {
		return nil, cerr
	}
	return ws, nil
}

func NewCommander(cfg *Config, listener NotificationListener) *Commander {
	c := &Commander{
		cfg:        cfg,
		listener:   listener,
		lastJSONID: 1,
	}
	return c
}

// Start dials to create and rpcConn and automatically reconnects to BTCD
func (c *Commander) Start() {
	// start up jsonID generator
	c.newJSONID = make(chan uint64)
	go func() {
		for {
			c.newJSONID <- c.lastJSONID
			c.lastJSONID++
		}
	}()

	c.running = true
	c.accessRpcConn = make(chan *AccessCurrentRpcConn)
	updateBtcd := make(chan *rpcConn)
	// run goroutine that controls access to Commander's current rpc
	go func() {
		c.rpc = nil
		var done chan struct{}
		for {
			select {
			case r := <-updateBtcd:
				c.rpc = r
				done = c.rpc.closed
			case access := <-c.accessRpcConn:
				access.result <- c.rpc
			case <-done:
				break
			}
		}
	}()

	finishedInit := make(chan struct{})
	// run reconnect loop
	go func() {
		for {
			btcd, err := newBtcdRpcConn(c.cfg, c.listener)
			if err != nil {
				log.Info("Retrying btcd connection in 5 seconds")
				time.Sleep(5 * time.Second)
				continue
			}
			updateBtcd <- btcd

			c.listener.BtcdConnected()
			log.Info("Established connection to btcd")

			// Perform handshake.
			if err := c.handshake(); err != nil {
				var message string
				if jsonErr, ok := err.(*btcjson.Error); ok {
					message = jsonErr.Message
				} else {
					message = err.Error()
				}
				log.Errorf("Cannot complete handshake: %v", message)
				log.Info("Retrying btcd connection in 5 seconds")
				time.Sleep(5 * time.Second)
				continue
			}
			log.Infof("Handshake Complete")

			if finishedInit != nil {
				close(finishedInit)
				finishedInit = nil
			}

			// Block goroutine until the connection is lost.
			<-btcd.closed
			c.listener.BtcdDisconnected()
			log.Info("Lost btcd connection")
			if !c.running {
				break
			}
		}
	}()

	<-finishedInit
}

// Stop terminates current connection and the reconnect loop to BTCD
func (c *Commander) Stop() {
	c.running = false

	rpcConn := c.currentRpcConn()
	if rpcConn != nil && rpcConn.connected() {
		rpcConn.ws.Close()
		close(rpcConn.closed)
	}
}

type AccessCurrentRpcConn struct {
	result chan *rpcConn
}

// can return nil
func (c *Commander) currentRpcConn() *rpcConn {
	access := &AccessCurrentRpcConn{
		result: make(chan *rpcConn),
	}
	c.accessRpcConn <- access
	return <-access.result
}

// Handshake first checks that the websocket connection between btcwallet and
// btcd is valid, that is, that there are no mismatching settings between
// the two processes (such as running on different Bitcoin networks).  If the
// sanity checks pass, all wallets are set to be tracked against chain
// notifications from this btcd connection.
func (c *Commander) handshake() error {
	net, jsonErr := c.GetCurrentNet()
	if jsonErr != nil {
		return jsonErr
	}
	if net != c.cfg.Net() {
		return errors.New("btcd and btcwallet running on different Bitcoin networks")
	}

	// Request notifications for connected and disconnected blocks.
	jsonErr = c.NotifyBlocks()
	if jsonErr != nil {
		return jsonErr
	}

	return nil
}

// GetCurrentNet requests the network a bitcoin RPC server is running on.
func (c *Commander) GetCurrentNet() (btcwire.BitcoinNet, *btcjson.Error) {
	rpc := c.currentRpcConn()
	cmd := btcws.NewGetCurrentNetCmd(<-c.newJSONID)
	response := <-rpc.sendRequest(cmd, nil)
	if response.err != nil {
		return 0, response.err
	}
	return btcwire.BitcoinNet(uint32(response.result.(float64))), nil
}

// NotifyBlocks requests blockconnected and blockdisconnected notifications.
func (c *Commander) NotifyBlocks() *btcjson.Error {
	rpc := c.currentRpcConn()
	cmd := btcws.NewNotifyBlocksCmd(<-c.newJSONID)
	response := <-rpc.sendRequest(cmd, nil)
	return response.err
}

type GetBestBlockResult struct {
	Hash   string `json:"hash"`
	Height int32  `json:"height"`
}

// GetBestBlock gets both the block height and hash of the best block
// in the main chain.
func (c *Commander) GetBestBlock() (*GetBestBlockResult, *btcjson.Error) {
	rpc := c.currentRpcConn()
	cmd := btcws.NewGetBestBlockCmd(<-c.newJSONID)
	response := <-rpc.sendRequest(cmd, new(GetBestBlockResult))
	if response.err != nil {
		return nil, response.err
	}
	return response.result.(*GetBestBlockResult), nil
}

// GetBlockHash gets the hash of the block at the given height
// in the main chain.
func (c *Commander) GetBlockHash(height int64) (string, *btcjson.Error) {
	rpc := c.currentRpcConn()
	cmd, err := btcjson.NewGetBlockHashCmd(<-c.newJSONID, height)
	if err != nil {
		return "", &btcjson.Error{
			Code:    -1,
			Message: err.Error(),
		}
	}
	response := <-rpc.sendRequest(cmd, nil)
	if response.err != nil {
		return "", response.err
	}
	return response.result.(string), nil
}

// GetBlock requests details about a block with the given hash.
func (c *Commander) GetBlock(blockHash string, verbose, verboseTx bool) (*btcjson.BlockResult, *btcjson.Error) {
	rpc := c.currentRpcConn()

	// NewGetBlockCmd cannot fail with no optargs, so omit the check.
	cmd, err := btcjson.NewGetBlockCmd(<-c.newJSONID, blockHash, verbose, verboseTx)
	if err != nil {
		return nil, &btcjson.Error{
			Code:    -1,
			Message: err.Error(),
		}
	}
	response := <-rpc.sendRequest(cmd, new(btcjson.BlockResult))
	if response.err != nil {
		return nil, response.err
	}
	return response.result.(*btcjson.BlockResult), nil
}
