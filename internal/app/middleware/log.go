package middleware

import (
	"github.com/GabrielHCataldo/go-logger/logger"
	"github.com/GabrielHCataldo/gopen-gateway/internal/app/interfaces"
	"github.com/GabrielHCataldo/gopen-gateway/internal/app/util"
	"github.com/gin-gonic/gin"
	"time"
)

type log struct {
	logProvider interfaces.LogProvider
}

type Log interface {
	Do(ctx *gin.Context)
}

func NewLog(logProvider interfaces.LogProvider) Log {
	return log{
		logProvider: logProvider,
	}
}

func (l log) Do(ctx *gin.Context) {
	// mantemos o tempo que a requisição começou
	startTime := time.Now()

	// inicializamos a logger options global, com o traceId e XForwardedFor
	l.logProvider.InitializeLoggerOptions(ctx)

	// imprimimos o log de start
	logger.Info("Start!", l.logProvider.BuildInitialRequestMessage(ctx))

	// chamamos o próximo handler da requisição
	ctx.Next()

	// imprimimos o log de finish
	logger.Info("Finish!", l.logProvider.BuildFinishRequestMessage(util.GetResponseWriter(ctx), startTime))
}