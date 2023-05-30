/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package global provides global configurations that can override others.
package global

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/webmeshproj/node/pkg/services"
	"github.com/webmeshproj/node/pkg/store"
	"github.com/webmeshproj/node/pkg/store/streamlayer"
	"github.com/webmeshproj/node/pkg/util"
	"github.com/webmeshproj/node/pkg/wireguard"
)

const (
	GlobalLogLevelEnvVar               = "GLOBAL_LOG_LEVEL"
	GlobalTLSCertEnvVar                = "GLOBAL_TLS_CERT_FILE"
	GlobalTLSKeyEnvVar                 = "GLOBAL_TLS_KEY_FILE"
	GlobalTLACAEnvVar                  = "GLOBAL_TLS_CA_FILE"
	GlobalTLSClientCAEnvVar            = "GLOBAL_TLS_CLIENT_CA_FILE"
	GlobalMTLSEnvVar                   = "GLOBAL_MTLS"
	GlobalSkipVerifyHostnameEnvVar     = "GLOBAL_SKIP_VERIFY_HOSTNAME"
	GlobalInsecureEnvVar               = "GLOBAL_INSECURE"
	GlobalNoIPv4EnvVar                 = "GLOBAL_NO_IPV4"
	GlobalNoIPv6EnvVar                 = "GLOBAL_NO_IPV6"
	GlobalPrimaryEndpointEnvVar        = "GLOBAL_PRIMARY_ENDPOINT"
	GlobalEndpointsEnvVar              = "GLOBAL_ENDPOINTS"
	GlobalDetectEndpointsEnvVar        = "GLOBAL_DETECT_ENDPOINTS"
	GlobalDetectPrivateEndpointsEnvVar = "GLOBAL_DETECT_PRIVATE_ENDPOINTS"
	GlobalAllowRemoteDetectionEnvVar   = "GLOBAL_ALLOW_REMOTE_DETECTION"
	GlobalDetectIPv6EnvVar             = "GLOBAL_DETECT_IPV6"
)

// Options are the global options.
type Options struct {
	// LogLevel is the log level.
	LogLevel string `yaml:"log-level,omitempty" json:"log-level,omitempty" toml:"log-level,omitempty"`
	// TLSCertFile is the TLS certificate file.
	TLSCertFile string `yaml:"tls-cert-file,omitempty" json:"tls-cert-file,omitempty" toml:"tls-cert-file,omitempty"`
	// TLSKeyFile is the TLS key file.
	TLSKeyFile string `yaml:"tls-key-file,omitempty" json:"tls-key-file,omitempty" toml:"tls-key-file,omitempty"`
	// TLACAFile is the TLS CA file.
	TLSCAFile string `yaml:"tls-ca-file,omitempty" json:"tls-ca-file,omitempty" toml:"tls-ca-file,omitempty"`
	// TLSClientCAFile is the path to the TLS client CA file.
	// If empty, either TLSCAFile or the system CA pool is used.
	TLSClientCAFile string `yaml:"tls-client-ca-file,omitempty" json:"tls-client-ca-file,omitempty" toml:"tls-client-ca-file,omitempty"`
	// MTLS is true if mutual TLS is enabled.
	MTLS bool `yaml:"mtls,omitempty" json:"mtls,omitempty" toml:"mtls,omitempty"`
	// SkipVerifyHostname is true if the hostname should not be verified.
	SkipVerifyHostname bool `yaml:"skip-verify-hostname,omitempty" json:"skip-verify-hostname,omitempty" toml:"skip-verify-hostname,omitempty"`
	// Insecure is true if TLS should be disabled.
	Insecure bool `yaml:"insecure,omitempty" json:"insecure,omitempty" toml:"insecure,omitempty"`
	// NoIPv4 is true if IPv4 should be disabled.
	NoIPv4 bool `yaml:"no-ipv4,omitempty" json:"no-ipv4,omitempty" toml:"no-ipv4,omitempty"`
	// NoIPv6 is true if IPv6 should be disabled.
	NoIPv6 bool `yaml:"no-ipv6,omitempty" json:"no-ipv6,omitempty" toml:"no-ipv6,omitempty"`
	// PrimaryEndpoint is the preferred publicly routable address of this node.
	// Setting this value will override the store advertise address with its
	// configured listen port.
	PrimaryEndpoint string `yaml:"primary-endpoint,omitempty" json:"endpoint,omitempty" toml:"endpoint,omitempty"`
	// Endpoints are the additional publicly routable addresses of this node.
	// If PrimaryEndpoint is not set, it will be set to the first endpoint.
	// Setting this value will override the store advertise with its configured
	// listen port.
	Endpoints []string `yaml:"endpoints,omitempty" json:"endpoints,omitempty" toml:"endpoints,omitempty"`
	// DetectEndpoints is true if the endpoints should be detected.
	DetectEndpoints bool `yaml:"detect-endpoints,omitempty" json:"detect-endpoints,omitempty" toml:"detect-endpoints,omitempty"`
	// DetectPrivateEndpoints is true if private IP addresses should be included in detection.
	// This automatically enables DetectEndpoints.
	DetectPrivateEndpoints bool `yaml:"detect-private-endpoints,omitempty" json:"detect-private-endpoints,omitempty" toml:"detect-private-endpoints,omitempty"`
	// AllowRemoteDetection is true if remote detection is allowed.
	AllowRemoteDetection bool `yaml:"allow-remote-detection,omitempty" json:"allow-remote-detection,omitempty" toml:"allow-remote-detection,omitempty"`
	// DetectIPv6 is true if IPv6 addresses should be included in detection.
	DetectIPv6 bool `yaml:"detect-ipv6,omitempty" json:"detect-ipv6,omitempty" toml:"detect-ipv6,omitempty"`
}

// NewOptions creates new options.
func NewOptions() *Options {
	return &Options{
		LogLevel: "info",
	}
}

func (o *Options) BindFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.TLSCertFile, "global.tls-cert-file", util.GetEnvDefault(GlobalTLSCertEnvVar, ""),
		"The certificate file for TLS connections.")
	fs.StringVar(&o.TLSKeyFile, "global.tls-key-file", util.GetEnvDefault(GlobalTLSKeyEnvVar, ""),
		"The key file for TLS connections.")
	fs.StringVar(&o.TLSCAFile, "global.tls-ca-file", util.GetEnvDefault(GlobalTLACAEnvVar, ""),
		"The CA file for TLS connections.")
	fs.StringVar(&o.TLSClientCAFile, "global.tls-client-ca-file", util.GetEnvDefault(GlobalTLSClientCAEnvVar, ""),
		"The client CA file for TLS connections.")
	fs.BoolVar(&o.MTLS, "global.mtls", util.GetEnvDefault(GlobalMTLSEnvVar, "false") == "true",
		"Enable mutual TLS globally.")
	fs.BoolVar(&o.SkipVerifyHostname, "global.skip-verify-hostname", util.GetEnvDefault(GlobalSkipVerifyHostnameEnvVar, "false") == "true",
		"Disable hostname verification globally.")
	fs.BoolVar(&o.Insecure, "global.insecure", util.GetEnvDefault(GlobalInsecureEnvVar, "false") == "true",
		"Disable use of TLS globally.")
	fs.BoolVar(&o.NoIPv6, "global.no-ipv6", util.GetEnvDefault(GlobalNoIPv6EnvVar, "false") == "true",
		"Disable use of IPv6 globally.")
	fs.BoolVar(&o.NoIPv4, "global.no-ipv4", util.GetEnvDefault(GlobalNoIPv4EnvVar, "false") == "true",
		"Disable use of IPv4 globally.")
	fs.StringVar(&o.LogLevel, "global.log-level", util.GetEnvDefault(GlobalLogLevelEnvVar, "info"),
		"Log level (debug, info, warn, error)")

	fs.StringVar(&o.PrimaryEndpoint, "global.primary-endpoint", util.GetEnvDefault(GlobalPrimaryEndpointEnvVar, ""),
		`The preferred publicly routable address of this node. Setting this
value will override the address portion of the store advertise address. 
When detect-endpoints is true, this value will be the first address detected.`)

	fs.BoolVar(&o.DetectEndpoints, "global.detect-endpoints", util.GetEnvDefault(GlobalDetectEndpointsEnvVar, "false") == "true",
		"Detect potential endpoints from the local interfaces.")

	fs.BoolVar(&o.DetectPrivateEndpoints, "global.detect-private-endpoints", util.GetEnvDefault(GlobalDetectPrivateEndpointsEnvVar, "false") == "true",
		"Include private IP addresses in detection.")

	fs.BoolVar(&o.AllowRemoteDetection, "global.allow-remote-detection", util.GetEnvDefault(GlobalAllowRemoteDetectionEnvVar, "false") == "true",
		"Allow remote detection of endpoints.")

	fs.BoolVar(&o.DetectIPv6, "global.detect-ipv6", util.GetEnvDefault(GlobalDetectIPv6EnvVar, "false") == "true",
		"Detect IPv6 addresses. Default is to only detect IPv4.")
}

// Overlay overlays the global options onto the given option sets.
func (o *Options) Overlay(opts ...any) error {
	var primaryEndpoint netip.Addr
	var endpoints util.PrefixList
	var err error
	if o.PrimaryEndpoint != "" {
		primaryEndpoint, err = netip.ParseAddr(o.PrimaryEndpoint)
		if err != nil {
			return fmt.Errorf("failed to parse endpoint: %w", err)
		}
	}
	if o.DetectEndpoints || o.DetectPrivateEndpoints {
		endpoints, err = util.DetectEndpoints(context.Background(), util.EndpointDetectOpts{
			DetectIPv6:           o.DetectIPv6,
			DetectPrivate:        o.DetectPrivateEndpoints,
			AllowRemoteDetection: o.AllowRemoteDetection,
		})
		if err != nil {
			return fmt.Errorf("failed to detect endpoints: %w", err)
		}
		if len(endpoints) > 0 {
			if !primaryEndpoint.IsValid() {
				primaryEndpoint = endpoints[0].Addr()
				if len(endpoints) > 1 {
					endpoints = endpoints[1:]
				} else {
					endpoints = nil
				}
			}
		}
	}
	for _, opt := range opts {
		switch v := opt.(type) {
		case *store.Options:
			if !v.NoIPv4 {
				v.NoIPv4 = o.NoIPv4
			}
			if !v.NoIPv6 {
				v.NoIPv6 = o.NoIPv6
			}
			if primaryEndpoint.IsValid() {
				var raftPort, wireguardPort uint16
				for _, inOpts := range opts {
					if vopt, ok := inOpts.(*streamlayer.Options); ok {
						_, port, err := net.SplitHostPort(vopt.ListenAddress)
						if err != nil {
							return fmt.Errorf("failed to parse raft listen address: %w", err)
						}
						raftPortz, err := strconv.ParseUint(port, 10, 16)
						if err != nil {
							return fmt.Errorf("failed to parse raft listen address: %w", err)
						}
						raftPort = uint16(raftPortz)
					}
				}
				for _, inOpts := range opts {
					if vopt, ok := inOpts.(*wireguard.Options); ok {
						wireguardPort = uint16(vopt.ListenPort)
					}
				}
				if raftPort == 0 {
					raftPort = 9443
				}
				if wireguardPort == 0 {
					wireguardPort = 51820
				}
				if v.NodeEndpoint == "" {
					v.NodeEndpoint = primaryEndpoint.String()
				}
				if v.NodeWireGuardEndpoints == "" {
					var eps []string
					if primaryEndpoint.IsValid() {
						eps = append(eps, netip.AddrPortFrom(primaryEndpoint, uint16(wireguardPort)).String())
					}
					for _, endpoint := range endpoints {
						ep := netip.AddrPortFrom(endpoint.Addr(), uint16(wireguardPort)).String()
						if ep != v.NodeEndpoint {
							eps = append(eps, ep)
						}
					}
					v.NodeWireGuardEndpoints = strings.Join(eps, ",")
				}
				if v.AdvertiseAddress == "" {
					v.AdvertiseAddress = netip.AddrPortFrom(primaryEndpoint, uint16(raftPort)).String()
				}
			}
		case *services.Options:
			if !v.Insecure {
				v.Insecure = o.Insecure
			}
			if !v.MTLS {
				v.MTLS = o.MTLS
			}
			if !v.SkipVerifyHostname {
				v.SkipVerifyHostname = o.SkipVerifyHostname
			}
			if v.TLSCertFile == "" {
				v.TLSCertFile = o.TLSCertFile
			}
			if v.TLSKeyFile == "" {
				v.TLSKeyFile = o.TLSKeyFile
			}
			if v.TLSCAFile == "" {
				v.TLSCAFile = o.TLSCAFile
			}
			if v.TLSClientCAFile == "" {
				v.TLSClientCAFile = o.TLSClientCAFile
			}
			if v.EnableTURNServer {
				if v.TURNServerEndpoint == "" && primaryEndpoint.IsValid() {
					v.TURNServerEndpoint = fmt.Sprintf("stun:%s",
						net.JoinHostPort(primaryEndpoint.String(), strconv.Itoa(v.TURNServerPort)))
				}
				if v.TURNServerPublicIP == "" && primaryEndpoint.IsValid() {
					v.TURNServerPublicIP = primaryEndpoint.String()
				}
			}
		case *streamlayer.Options:
			if !v.Insecure {
				v.Insecure = o.Insecure
			}
			if !v.MTLS {
				v.MTLS = o.MTLS
			}
			if !v.SkipVerifyHostname {
				v.SkipVerifyHostname = o.SkipVerifyHostname
			}
			if v.TLSCertFile == "" {
				v.TLSCertFile = o.TLSCertFile
			}
			if v.TLSKeyFile == "" {
				v.TLSKeyFile = o.TLSKeyFile
			}
			if v.TLSCAFile == "" {
				v.TLSCAFile = o.TLSCAFile
			}
			if v.TLSClientCAFile == "" {
				v.TLSClientCAFile = o.TLSClientCAFile
			}
		}
	}
	return nil
}
