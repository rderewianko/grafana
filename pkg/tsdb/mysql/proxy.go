package mysql

import (
	"context"
	"net"

	"github.com/go-sql-driver/mysql"
	iproxy "github.com/grafana/grafana/pkg/infra/proxy"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util"
	"golang.org/x/net/proxy"
)

// registerProxyDialerContext registers a new dialer context to be used by mysql when the proxy network is
// specified in the connection string
func registerProxyDialerContext(settings *setting.SecureSocksDSProxySettings, protocol, cnnstr string) (string, error) {
	// the dialer contains the true network used behind the scenes
	dialer, err := getProxyDialerContext(settings, protocol)
	if err != nil {
		return "", err
	}

	// the dialer context can be updated everytime the datasource is updated
	// have a unique network per connection string
	hash, err := util.Md5SumString(cnnstr)
	if err != nil {
		return "", err
	}
	network := "proxy-" + hash
	mysql.RegisterDialContext(network, dialer.DialContext)

	return network, nil
}

// mySQLContextDialer turns a golang proxy driver into a MySQL proxy driver
type mySQLContextDialer struct {
	dialer  proxy.ContextDialer
	network string
}

// getProxyDialerContext returns a context dialer that will send the request through to the secure socks proxy
func getProxyDialerContext(cfg *setting.SecureSocksDSProxySettings, actualNetwork string) (*mySQLContextDialer, error) {
	dialer, err := iproxy.NewSecureSocksProxyContextDialer(cfg)
	if err != nil {
		return nil, err
	}
	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, err
	}
	return &mySQLContextDialer{dialer: contextDialer, network: actualNetwork}, nil
}

// DialContext implements the MySQL requirements for a proxy driver, and uses the underlying golang proxy driver with the assigned network
func (d *mySQLContextDialer) DialContext(ctx context.Context, addr string) (net.Conn, error) {
	return d.dialer.DialContext(ctx, d.network, addr)
}
