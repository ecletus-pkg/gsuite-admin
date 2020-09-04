package gsuite_admin

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/ecletus/auth"
	"github.com/ecletus/sites"
	"github.com/moisespsena-go/httpu"

	"github.com/ecletus-pkg/siteconf"

	"github.com/ecletus/core"

	admin_plugin "github.com/ecletus-pkg/admin"
	"github.com/ecletus/admin"
	"github.com/ecletus/db"
	"github.com/ecletus/ecletus"
	"github.com/ecletus/plug"
	"github.com/ecletus/router"
	"github.com/moisespsena-go/assetfs/assetfsapi"
	gsuite_admin_app "github.com/moisespsena-go/gsuite-admin-app"
	"github.com/moisespsena-go/http-render/ropt"
	"github.com/moisespsena-go/httpd"
	path_helpers "github.com/moisespsena-go/path-helpers"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger(path_helpers.GetCalledDir())

type Plugin struct {
	plug.EventDispatcher
	db.DBNames
	admin_plugin.AdminNames

	RouterKey,
	RouterUID,
	ConfigKey,
	AssetFSKey,
	AuthKey,
	SitesRegisterKey,
	GSuiteAdminAppKey string

	resProtector,
	resPasswordUpdater *admin.Resource
	app    *gsuite_admin_app.App
	server *httpd.Server
}

func (this *Plugin) RequireOptions() []string {
	return []string{this.SitesRegisterKey, this.AuthKey}
}

func (this *Plugin) ProvideOptions() []string {
	return []string{this.GSuiteAdminAppKey}
}

func (this *Plugin) Before() []string {
	return []string{this.RouterUID}
}

func (this *Plugin) OnRegister(options *plug.Options) {
	if this.RouterKey == "" {
		panic("RouterKey is BLANK")
	}

	admin_plugin.Events(this).InitResources(func(e *admin_plugin.AdminEvent) {
		e.Admin.OnResourceValueAdded(&siteconf.SiteConfigMain{}, func(e *admin.ResourceEvent) {
			e.Resource.Action(&admin.Action{
				Name:     "gsuite-admin-setup",
				LabelKey: i18ngroup + ".setup_action.Label",
				Handler: func(argument *admin.ActionArgument) error {
					w, r := argument.Context.Writer, argument.Context.Request
					s, err := gsuite_admin_app.GetRegisterSession(r)
					if err != nil {
						return err
					}

					if err := s.Delete(w); err != nil {
						return err
					}
					site := core.GetSiteFromRequest(r)
					if err := s.Set(w, "site-name", site.Name()); err != nil {
						return err
					}
					s.Data().RedirectTo = argument.Context.URL("gsuite-setup")
					s.Data().StoreTokenUrl = httpu.HttpScheme(r) + "://" + r.Host + sites.RootPath(r) + "/gsuite-app/register"
					if err := s.Save(w); err != nil {
						return err
					}
					w.Header().Set("X-Location", sites.RootPath(r)+"/gsuite-app/")
					argument.SkipDefaultResponse = true
					return nil
				},
			})
		})
	})
}

func (this *Plugin) ProvidesOptions(options *plug.Options) {
	cfg := options.GetInterface(this.ConfigKey).(*ecletus.ConfigDir)
	pth := cfg.Path("gsuite-admin-app", "credentials.json")
	r, err := os.Open(pth)
	if err != nil {
		log.Error(err)
		return
	}
	b, err := ioutil.ReadAll(r)
	this.app = gsuite_admin_app.New("my_customer")

	if err = this.app.LoadCredentials(b); err != nil {
		log.Error("load credentials failed: " + err.Error())
		return
	}
	//this.app.AddScope("https://apps-apis.google.com/a/feeds/domain/")

	this.app.DomainPageURL = strings.TrimSuffix(this.app.Crendentials.RedirectURL, "setup")
	this.app.TokenStorage = TokenStorage{
		this.app,
		options.GetInterface(this.SitesRegisterKey).(*core.SitesRegister),
	}
	this.server = httpd.New(nil)
	fs := options.GetInterface(this.AssetFSKey).(assetfsapi.Interface)
	this.app.IndexHandler = this.server.GetOrCreateRender().Option(ropt.FS(fs.NameSpace("templates/gsuite-admin-app"))).ServeHTTP
	appHandler := this.app.Handler()
	this.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appHandler.ServeHTTP(w, r)
	})

	options.Set(this.GSuiteAdminAppKey, this.app)
}

func (this *Plugin) Init(options *plug.Options) {
	router.OnRoute(this, func(e *router.RouterEvent) {
		e.Router.Mux.HandleFunc("/admin/gsuite-setup", func(w http.ResponseWriter, r *http.Request) {
			Auth := options.GetInterface(this.AuthKey).(*auth.Auth)
			auth.Authenticates(Auth, w, r, func(ok bool) {
				ctx := core.ContextFromRequest(r)
				domain := siteconf.MustPrivate(ctx.Site, DomainKey)
				tok, _ := this.app.TokenStorage.Get(r, domain)
				if err := this.app.Setup(tok, r); err != nil {
					http.Error(w, "gsuite setup failed: "+err.Error(), http.StatusInternalServerError)
					return
				}
				if !ctx.FlashTOrError(MsgSetupSuccessful, "info") {
					return
				}
				http.Redirect(w, r, ctx.Path(), http.StatusSeeOther)
			})
		})
		e.Router.PrefixHandlers.With("/gsuite-app/", this.server)
	})
}
