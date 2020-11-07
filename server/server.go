package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/cement/x509"
	"github.com/zdnscloud/gorest"
	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/gorest/resource/schema"

	"github.com/trymanytimes/UpdateWeb/config"
	"github.com/trymanytimes/UpdateWeb/pkg/auth/authentification"
	"github.com/trymanytimes/UpdateWeb/pkg/util"
)

const (
	defaultTlsCertFile = "tls_cert.crt"
	defaultTlsKeyFile  = "tls_key.key"
)

type Server struct {
	group     *gin.RouterGroup
	router    *gin.Engine
	apiServer *gorest.Server
}

type HandlerRegister func(*gorest.Server, gin.IRoutes) error

func (h HandlerRegister) RegisterHandler(server *gorest.Server, router gin.IRoutes) error {
	return h(server, router)
}

type WebHandler interface {
	RegisterHandler(*gorest.Server, gin.IRoutes) error
}

func NewServer() (*Server, error) {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = os.Stdout
	router := gin.New()
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] client:%s \"%s %s\" %s %d %s %s\n",
			param.TimeStamp.Format(util.TimeFormat),
			param.ClientIP,
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
		)
	}))
	router.StaticFS("/public", http.Dir(util.FileRootPath))
	router.POST("/login", authentification.Login)
	router.GET("/", func(context *gin.Context) {
		context.Redirect(http.StatusFound, "/public")
	})
	group := router.Group("/")
	apiServer := gorest.NewAPIServer(schema.NewSchemaManager())
	apiServer.Use(authentification.JWTMiddleWare())
	return &Server{
		group:     group,
		router:    router,
		apiServer: apiServer,
	}, nil
}

func (s *Server) RegisterHandler(h WebHandler) error {
	return h.RegisterHandler(s.apiServer, s.router)
}

func (s *Server) Run(conf *config.DDIControllerConfig) error {
	adaptor.RegisterHandler(s.group, s.apiServer, s.apiServer.Schemas.GenerateResourceRoute())
	if conf.Server.TlsCertFile == "" {
		if err := createSelfSignedTlsCert(); err != nil {
			return err
		}

		return s.router.RunTLS(":"+conf.Server.Port, defaultTlsCertFile, defaultTlsKeyFile)
	} else {
		return s.router.RunTLS(":"+conf.Server.Port, conf.Server.TlsCertFile, conf.Server.TlsKeyFile)
	}
}

func createSelfSignedTlsCert() error {
	_, err := os.Stat(defaultTlsCertFile)
	if err != nil && os.IsExist(err) {
		return nil
	}

	cert, err := x509.GenerateSelfSignedCertificate("ddi.linkingthing.com", nil, nil, 7300)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(defaultTlsCertFile, []byte(cert.Cert), 0644); err != nil {
		return err
	}
	return ioutil.WriteFile(defaultTlsKeyFile, []byte(cert.Key), 0644)
}
