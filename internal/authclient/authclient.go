package authclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/sploders101/personal-website/internal/env"
	authv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/auth/v1"
	"github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/auth/v1/authv1connect"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var ErrNoKeys = errors.New("no valid ssh keys found")
var ErrProtocolViolation = errors.New("protocol violation")

// TODO: Source these from other means
const TOKEN_EXPIRATION = 10 * time.Minute

type Authenticator struct {
	client     authv1connect.AuthServiceClient
	expiration time.Time
	token      string
}

func NewAuthenticator(endpoint string) *Authenticator {
	client := authv1connect.NewAuthServiceClient(env.ConnectHttp, endpoint)

	return &Authenticator{
		client: client,
	}
}

func (authenticator *Authenticator) Authenticate(ctx context.Context) (string, error) {
	if authenticator.expiration.Before(time.Now()) {
		return authenticator.AuthenticateSSH(ctx)
	}
	return authenticator.token, nil
}

func (authenticator *Authenticator) AuthenticateSSH(ctx context.Context) (string, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return "", ErrNoKeys
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return "", fmt.Errorf("failed to connect to ssh agent: %w", err)
	}

	myAgent := agent.NewClient(conn)

	keys, err := myAgent.List()
	if err != nil {
		return "", fmt.Errorf("failed to list ssh keys: %w", err)
	}

	permit, err := authenticator.client.GetAuthPermit(ctx, authv1.GetAuthPermitRequest_builder{}.Build())
	if err != nil {
		return "", fmt.Errorf("failed to get auth permit: %w", err)
	}

	for _, key := range keys {
		fingerprint := ssh.FingerprintSHA256(key)

		// Check if fingerprint is acceptable
		checkResp, err := authenticator.client.CheckSSHKey(ctx, authv1.CheckSSHKeyRequest_builder{
			Permit: permit.GetPermit(),
			Fingerprint: fingerprint,
		}.Build())
		if err != nil {
			return "", fmt.Errorf("failed to check SSH key: %w", err)
		}

		if !checkResp.GetRecognized() {
			continue
		}

		signed, err := myAgent.Sign(key, []byte(permit.GetPermit()))
		if err != nil {
			return "", err
		}
		exchangeResp, err := authenticator.client.ExchangeSSHPermit(ctx, authv1.ExchangeSSHPermitRequest_builder{
			Permit: permit.GetPermit(),
			Fingerprint: fingerprint,
			Format: signed.Format,
			Signature: signed.Blob,
		}.Build())
		if err != nil {
			return "", err
		}

		authToken := exchangeResp.GetToken()
		authenticator.token = authToken
		authenticator.expiration = time.Now().Add(TOKEN_EXPIRATION)
		return authToken, nil
	}

	return "", errors.New("no keys accepted")
}

func (authenticator *Authenticator) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		token, err := authenticator.Authenticate(ctx)
		if err != nil {
			return nil, connect.NewError(
				connect.CodeUnauthenticated,
				fmt.Errorf("failed authentication: %w", err),
			)
		}
		req.Header().Set("Authorization", "Bearer "+token)
		return next(ctx, req)
	}
}

func (authenticator *Authenticator) WrapStreamingClient(
	next connect.StreamingClientFunc,
) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		token, err := authenticator.Authenticate(ctx)
		if err != nil {
			return &authErrorConn{
				err: connect.NewError(
					connect.CodeUnauthenticated,
					fmt.Errorf("failed authentication: %w", err),
				),
			}
		}
		conn := next(ctx, spec)
		conn.RequestHeader().Add("Authorization", "Bearer "+token)
		return conn
	}
}

func (authenticator *Authenticator) WrapStreamingHandler(
	next connect.StreamingHandlerFunc,
) connect.StreamingHandlerFunc {
	return next
}

type authErrorConn struct {
	err error
}

func (c *authErrorConn) Spec() connect.Spec {
	return connect.Spec{}
}

func (c *authErrorConn) Peer() connect.Peer {
	return connect.Peer{}
}

func (c *authErrorConn) Send(any) error {
	return c.err
}

func (c *authErrorConn) Receive(any) error {
	return c.err
}

func (c *authErrorConn) RequestHeader() http.Header {
	return make(http.Header)
}

func (c *authErrorConn) ResponseHeader() http.Header {
	return make(http.Header)
}

func (c *authErrorConn) ResponseTrailer() http.Header {
	return make(http.Header)
}

func (c *authErrorConn) CloseRequest() error {
	return c.err
}

func (c *authErrorConn) CloseResponse() error {
	return c.err
}
