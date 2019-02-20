package liveconfig

import (
	"context"
	"github.com/kyawmyintthein/liveconfig/option"
	"time"

)

type hostsKey struct{}
type usernameKey struct{}
type passwordKey struct{}
type dealTimeoutKey struct{}
type requestTimeoutKey struct{}
type filepathsKey struct{}
type configTypeKey struct{}

// WithHosts sets the etcd hosts option
func WithHosts(hosts ...string) option.Option {
	return func(o *option.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, hostsKey{}, hosts)
	}
}

// WithUsername sets the etcd username option
func WithUsername(username string) option.Option {
	return func(o *option.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, usernameKey{}, username)
	}
}

// WithPassword sets the etcd password option
func WithPassword(password string) option.Option {
	return func(o *option.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, passwordKey{}, password)
	}
}

// WithDialtimeout sets the dial timeout option
func WithDialtimeout(dialTimeout time.Duration) option.Option {
	return func(o *option.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, dealTimeoutKey{}, dialTimeout)
	}
}

// WithRequesttimeout sets the request timeout option
func WithRequesttimeout(requestTimeout time.Duration) option.Option {
	return func(o *option.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, requestTimeoutKey{}, requestTimeout)
	}
}


// WithFilepaths sets the filepaths for viper option
func WithFilepaths(filepaths []string) option.Option {
	return func(o *option.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, filepathsKey{}, filepaths)
	}
}


// WithConfigType sets the config type option
func WithConfigType(configType string) option.Option {
	return func(o *option.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, configTypeKey{}, configType)
	}
}
