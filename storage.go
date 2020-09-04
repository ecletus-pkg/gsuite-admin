package gsuite_admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/moisespsena-go/i18n-modular/i18nmod"

	"github.com/ecletus-pkg/siteconf"
	"github.com/ecletus/core"
	"github.com/moisespsena-go/getters"
	gsuite_admin_app "github.com/moisespsena-go/gsuite-admin-app"
	path_helpers "github.com/moisespsena-go/path-helpers"
)

var (
	pkg       = path_helpers.GetCalledDir()
	i18ngroup = i18nmod.PkgToGroup(pkg)
	key       = siteconf.PrivateName(pkg)
	tokenKey  = key.Sub("token")
	DomainKey = key.Sub("domain")
)

func DomainTokenKey(domain string) siteconf.PrivateName {
	return tokenKey.Concat("@" + domain)
}

type TokenKey struct{}

type TokenStorage struct {
	app   *gsuite_admin_app.App
	sites *core.SitesRegister
}

func (this TokenStorage) Get(r *http.Request, domain string) (token *gsuite_admin_app.Token, err error) {
	var session gsuite_admin_app.RegisterSession
	if err = session.Load(r); err != nil {
		return
	}

	var site *core.Site
	if site = core.GetSiteFromRequest(r);site == nil {
		site, _ = this.sites.Get(session.GetS("site-name"))
	}
	s, _ := getters.String(site.Config(), DomainTokenKey(domain))
	if s == "" {
		err = os.ErrNotExist
		return
	}
	token = &gsuite_admin_app.Token{}
	if err = json.NewDecoder(bytes.NewBufferString(s)).Decode(token); err != nil {
		token = nil
		return
	}
	return
}

func (this TokenStorage) Put(r *http.Request, token *gsuite_admin_app.Token) (err error) {
	var session gsuite_admin_app.RegisterSession
	if err = session.Load(r); err != nil {
		return
	}
	site, _ := this.sites.Get(session.GetS("site-name"))
	if site == nil {
		return errors.New("site-name is blank")
	}
	b, _ := json.Marshal(token)
	err = siteconf.SetPrivateMap(site, map[interface{}]interface{}{
		DomainTokenKey(token.Domain): string(b),
		DomainKey:                    token.Domain,
	})
	return
}
