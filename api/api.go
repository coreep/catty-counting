package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/log"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type Api struct {
	deps deps.Deps
}

func NewApi(deps deps.Deps) *Api {
	deps.Logger = deps.Logger.With(log.CALLER, "api.api")
	return &Api{deps: deps}
}

func (a *Api) Run(ctx context.Context) {
	router := gin.New()
	router.Use(a.logging(), gin.Recovery())

	router.GET("/api/file/:key", a.provideFile)

	a.deps.Logger.Info("Starting API server on port " + config.ApiPort())
	// TODO: handle graceful shutdown; context
	if err := router.Run(":" + config.ApiPort()); err != nil {
		a.deps.Logger.With(log.ERROR, errors.WithStack(err)).Error("Api failed")
	}
	a.deps.Logger.Debug("api done")
}

func (a *Api) provideFile(c *gin.Context) {
	key := c.Param("key")
	logger := a.deps.Logger.With(slog.String("key", key))

	var exposedFile db.ExposedFile
	if err := a.deps.DBC.WithContext(c.Request.Context()).Joins("File").First(&exposedFile, "key = ?", key).Error; err != nil {
		logger.With(log.ERROR, err).Error("failed to find exposed file")
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	logger = logger.With(slog.String("blob_key", exposedFile.File.BlobKey))

	reader, err := a.deps.Files.NewReader(c.Request.Context(), exposedFile.File.BlobKey, nil)
	if err != nil {
		logger.With(log.ERROR, err).Error("failed to read from blob")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	defer func() {
		if err := reader.Close(); err != nil {
			a.deps.Logger.With(log.ERROR, errors.WithStack(err)).Warn("failed to close blob reader")
		}
	}()

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", exposedFile.File.OriginalName))
	c.Header("Content-Type", exposedFile.File.MimeType)
	c.Header("Content-Length", fmt.Sprintf("%d", exposedFile.File.Size))

	if _, err := io.Copy(c.Writer, reader); err != nil {
		logger.With(log.ERROR, err).Error("failed to copy file to response")
		return
	}
}

func (a *Api) logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		param := gin.LogFormatterParams{
			Request: c.Request,
			Keys:    c.Keys,
		}

		param.TimeStamp = time.Now()
		param.Latency = param.TimeStamp.Sub(start)
		param.ClientIP = c.ClientIP()
		param.Method = c.Request.Method
		param.StatusCode = c.Writer.Status()
		param.ErrorMessage = c.Errors.ByType(gin.ErrorTypePrivate).String()
		param.BodySize = c.Writer.Size()
		if raw != "" {
			path = path + "?" + raw
		}
		param.Path = path

		a.deps.Logger.
			With("time", param.TimeStamp).
			With("latency", param.Latency).
			With("client_ip", param.ClientIP).
			With("method", param.Method).
			With("status_code", param.StatusCode).
			With("error_message", param.ErrorMessage).
			With("body_size", param.BodySize).
			With("path", param.Path).
			Info("request")
	}
}
