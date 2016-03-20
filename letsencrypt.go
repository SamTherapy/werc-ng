package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"net"
	"strings"

	"github.com/dkumor/acmewrapper"
)

var (
	letsencrypt = flag.Bool("letsencrypt", false, "enable Let’s Encrypt")
	domainname  = flag.String("domain", "", "domain for certificate")
	sannames    = flag.String("sans", "", "comma-seperate list of subject alternate names for certificate")
	certfile    = flag.String("cert", "", "TLS cert file path")
	keyfile     = flag.String("key", "", "TLS key file path")
	regfile     = flag.String("reg", "", "user registration file path")
	privfile    = flag.String("priv", "", "user private key file path")
	test        = flag.Bool("test", false, "Use the Let's Encrypt staging server")
	acme        = flag.String("server", acmewrapper.DefaultServer, "The ACME server to use")
	accept      = flag.Bool("accept", false, "Accept the ACME server's TOS?")
	email       = flag.String("email", "", "The email to use when registering")

	ErrDisabled = errors.New("Let's Encrypt is disabled")
)

// setup a tls listener and config from let's encrypt settings.
//
// TODO(mischief): http redirect to https
func doTLS(addr string) (net.Listener, *tls.Config, error) {
	if *letsencrypt == false {
		return nil, nil, ErrDisabled
	}

	if *test {
		*acme = "https://acme-staging.api.letsencrypt.org/directory"
	}

	domains := []string{*domainname}

	if *sannames != "" {
		sans := strings.Split(*sannames, ",")
		domains = append(domains, sans...)
	}

	aconf := acmewrapper.Config{
		Address:          addr,
		Domains:          domains,
		Email:            *email,
		TLSCertFile:      *certfile,
		TLSKeyFile:       *keyfile,
		RegistrationFile: *regfile,
		PrivateKeyFile:   *privfile,
		Server:           *acme,
		TOSCallback:      acmewrapper.TOSAgree,
	}

	if *accept == false {
		aconf.TOSCallback = acmewrapper.TOSDecline
	}

	w, err := acmewrapper.New(aconf)
	if err != nil {
		return nil, nil, err
	}

	tlsconfig := w.TLSConfig()

	listener, err := tls.Listen("tcp", addr, tlsconfig)
	if err != nil {
		return nil, nil, err
	}

	return listener, tlsconfig, nil
}
