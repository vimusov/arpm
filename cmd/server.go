package main

/*
   This file is part of arpm.

   arpm is free software: you can redistribute it and/or modify it under the terms
   of the GNU General Public License as published by the Free Software Foundation, either
   version 3 of the License, or (at your option) any later version.

   arpm is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
   without even the implied warranty     of MERCHANTABILITY or FITNESS FOR A PARTICULAR
   PURPOSE. See the GNU General Public License for more details.

   You should have received a copy of the GNU General Public License along with arpm.
   If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const notifySocket = "NOTIFY_SOCKET"

func notifyReady() {
	addr := &net.UnixAddr{Name: os.Getenv(notifySocket), Net: "unixgram"}
	if addr.Name == "" {
		return
	}
	_ = os.Unsetenv(notifySocket)
	conn, dialErr := net.DialUnix(addr.Net, nil, addr)
	if dialErr != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	_, _ = conn.Write([]byte("READY=1"))
}

func runServer(rootDir string) error {
	engine := echo.New()
	engine.HidePort = true
	engine.HideBanner = true

	if debugMode {
		engine.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Format: "${method} ${uri}: ${status} ${error}.\n",
		}))
	}

	engine.GET("/branches", func(c echo.Context) error { return lsBranchesHandler(rootDir, c) })
	engine.POST("/branches", func(c echo.Context) error { return addBranchHandler(rootDir, c) })

	engine.GET("/packages/:branch", func(c echo.Context) error { return lsPkgsHandler(rootDir, c) })
	engine.POST("/packages/:branch", func(c echo.Context) error { return addPkgHandler(rootDir, c) })
	engine.DELETE("/packages/:branch", func(c echo.Context) error { return rmPkgHandler(rootDir, c) })

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		notifyReady()
		if srvErr := engine.Start("127.0.0.1:31847"); srvErr != nil && srvErr != http.ErrServerClosed {
			engine.Logger.Fatal("failed to shutdown server")
		}
	}()

	<-signals

	if closeErr := engine.Close(); closeErr != nil {
		engine.Logger.Fatal(closeErr)
	}

	wg.Wait()
	return nil
}
