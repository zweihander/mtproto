package mtproto

import (
	"github.com/xelaj/mtproto/serialize"
)

type ReqPQParams struct {
	Nonce *serialize.Int128
}

func (_ *ReqPQParams) CRC() uint32 { return 0x60469778 }

func (m *MTProto) ReqPQ(nonce *serialize.Int128) (*serialize.ResPQ, error) {
	pq := new(serialize.ResPQ)
	if err := m.MakeRequest2(&ReqPQParams{Nonce: nonce}, pq); err != nil {
		return nil, err
	}

	return pq, nil
}

type ReqDHParamsParams struct {
	Nonce                *serialize.Int128
	ServerNonce          *serialize.Int128
	P                    []byte
	Q                    []byte
	PublicKeyFingerprint int64
	EncryptedData        []byte
}

func (_ *ReqDHParamsParams) CRC() uint32 {
	return 0xd712e4be
}

func (m *MTProto) ReqDHParams(nonce, serverNonce *serialize.Int128, p, q []byte, publicKeyFingerprint int64, encryptedData []byte) (serialize.ServerDHParams, error) {
	var dhp serialize.ServerDHParams
	if err := m.MakeRequest2(&ReqDHParamsParams{
		Nonce:                nonce,
		ServerNonce:          serverNonce,
		P:                    p,
		Q:                    q,
		PublicKeyFingerprint: publicKeyFingerprint,
		EncryptedData:        encryptedData,
	}, &dhp); err != nil {
		return nil, err
	}

	return dhp, nil
}

type SetClientDHParamsParams struct {
	Nonce         *serialize.Int128
	ServerNonce   *serialize.Int128
	EncryptedData []byte
}

func (_ *SetClientDHParamsParams) CRC() uint32 {
	return 0xf5045f1f
}

func (m *MTProto) SetClientDHParams(nonce, serverNonce *serialize.Int128, encryptedData []byte) (serialize.SetClientDHParamsAnswer, error) {
	var dhpa serialize.SetClientDHParamsAnswer
	if err := m.MakeRequest2(&SetClientDHParamsParams{
		Nonce:         nonce,
		ServerNonce:   serverNonce,
		EncryptedData: encryptedData,
	}, &dhpa); err != nil {
		return nil, err
	}

	return dhpa, nil
}

// rpc_drop_answer
// get_future_salts

type PingParams struct {
	PingID int64
}

func (_ *PingParams) CRC() uint32 {
	return 0x7abe77ec
}

func (m *MTProto) Ping(pingID int64) (*serialize.Pong, error) {
	pong := new(serialize.Pong)
	if err := m.MakeRequest2(&PingParams{
		PingID: pingID,
	}, pong); err != nil {
		return nil, err
	}

	return pong, nil
}

// ping_delay_disconnect
// destroy_session
// http_wait

// set_client_DH_params#f5045f1f nonce:int128 server_nonce:int128 encrypted_data:bytes = Set_client_DH_params_answer;

// rpc_drop_answer#58e4a740 req_msg_id:long = RpcDropAnswer;
// get_future_salts#b921bd04 num:int = FutureSalts;
// ping_delay_disconnect#f3427b8c ping_id:long disconnect_delay:int = Pong;
// destroy_session#e7512126 session_id:long = DestroySessionRes;

// http_wait#9299359f max_delay:int wait_after:int max_wait:int = HttpWait;
