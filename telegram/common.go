package telegram

import (
	"crypto/rsa"
	"fmt"
	"runtime"

	"github.com/k0kubun/pp"
	"github.com/pkg/errors"

	"github.com/xelaj/mtproto"
	"github.com/xelaj/mtproto/encoding/tl"
)

const ApiVersion = 117

type Client struct {
	*mtproto.MTProto
	config *ClientConfig
}

type ClientConfig struct {
	SessionStore  mtproto.SessionStore
	ServerHost    string
	DeviceModel   string
	SystemVersion string
	AppVersion    string
	AppID         int
	AppHash       string
}

func NewClient(host string, pubKey *rsa.PublicKey, cfg ClientConfig) (*Client, error) {
	if cfg.DeviceModel == "" {
		cfg.DeviceModel = "Unknown"
	}

	if cfg.SystemVersion == "" {
		cfg.SystemVersion = runtime.GOOS + "/" + runtime.GOARCH
	}

	if cfg.AppVersion == "" {
		cfg.AppVersion = "v0.0.0"
	}

	m, err := mtproto.NewMTProto(host, pubKey, cfg.SessionStore)
	if err != nil {
		return nil, errors.Wrap(err, "setup common MTProto client")
	}
	fmt.Println("mtproto created")

	client := &Client{
		MTProto: m,
		config:  &cfg,
	}

	client.AddCustomServerRequestHandler(client.handleSpecialRequests())
	fmt.Println("HelpGetCfgParams invoking...")
	config := new(Config)
	err = client.InvokeWithLayer(ApiVersion, &InitConnectionParams{
		ApiID:          int32(cfg.AppID),
		DeviceModel:    cfg.DeviceModel,
		SystemVersion:  cfg.SystemVersion,
		AppVersion:     cfg.AppVersion,
		SystemLangCode: "en", // can't be edited, cause docs says that a single possible parameter
		LangCode:       "en",
		Query:          &HelpGetConfigParams{},
	}, config)
	fmt.Println("HelpGetCfgParams done...")
	if err != nil {
		return nil, errors.Wrap(err, "getting server configs")
	}

	pp.Println(config)

	return client, nil
}

func (c *Client) handleSpecialRequests() func(interface{}) bool {
	return func(i interface{}) bool {
		switch msg := i.(type) {
		case *UpdatesObj:
			pp.Println(msg, "UPDATE")
			return true
		case *UpdateShort:
			pp.Println(msg, "SHORT UPDATE")
			return true
		}

		return false
	}
}

//----------------------------------------------------------------------------

type InvokeWithLayerParams struct {
	Layer int32
	Query tl.Object
}

func (_ *InvokeWithLayerParams) CRC() uint32 { return 0xda9b0d0d }

func (m *Client) InvokeWithLayer(layer int, query tl.Object, resp interface{}) error {
	return m.MakeRequest(&InvokeWithLayerParams{
		Layer: int32(layer),
		Query: query,
	}, resp)
}

type InvokeWithTakeoutParams struct {
	TakeoutID int64
	Query     tl.Object
}

func (*InvokeWithTakeoutParams) CRC() uint32 { return 0xaca9fd2e }

func (m *Client) InvokeWithTakeout(takeoutID int, query tl.Object, resp interface{}) error {
	return m.MakeRequest(&InvokeWithTakeoutParams{
		TakeoutID: int64(takeoutID),
		Query:     query,
	}, resp)
}

type InitConnectionParams struct {
	ApiID          int32
	DeviceModel    string
	SystemVersion  string
	AppVersion     string
	SystemLangCode string
	LangPack       string
	LangCode       string
	Proxy          *InputClientProxy `tl:"flag:0"`
	Params         JSONValue         `tl:"flag:1"`
	Query          tl.Object
}

func (_ *InitConnectionParams) CRC() uint32 { return 0xc1cd5ea9 }
