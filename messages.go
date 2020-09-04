package gsuite_admin

import (
	"strings"

	"github.com/moisespsena-go/i18n-modular/i18nmod"
)

func i18nkey(key, category string) string {
	return strings.Replace(i18ngroup+"."+category+"."+key, " ", "_", 1)
}

type msg string

func (this msg) Translate(ctx i18nmod.Context) string {
	return ctx.T(i18nkey(string(this), "messages")).Get()
}

type err string

func (this err) Translate(ctx i18nmod.Context) string {
	return ctx.T(i18nkey(string(this), "errors")).Get()
}

func (this err) Error() string {
	return string(this)
}

const (
	MsgSetupSuccessful msg = "setup_successful"
)
