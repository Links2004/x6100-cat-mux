package main

import (
	"io"
	"net"
	"os"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func main() {
	var loggerConfig = zap.NewDevelopmentConfig()
	loggerConfig.Level.SetLevel(zap.DebugLevel)

	logger, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}

	APPconn, err := net.Dial("tcp", "127.0.0.1:9990")
	if err != nil {
		logger.Error("error connection to socat", zap.Error(err))
		return
	}
	defer APPconn.Close()

	logger.Info("connected to APP", zap.String("addr", APPconn.RemoteAddr().String()))

	l, err := net.Listen("tcp", "127.0.0.1:9090")
	if err != nil {
		return
	}

	defer l.Close()

	var connMap = &sync.Map{}

	go appRX(connMap, logger, APPconn)

	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Error("error accepting connection", zap.Error(err))
			return
		}

		id := uuid.New().String()

		logger.Info("client connected", zap.String("id", id), zap.String("addr", conn.RemoteAddr().String()))
		connMap.Store(id, conn)

		go appTX(id, conn, connMap, logger, APPconn)
	}

}

func appRX(connMap *sync.Map, logger *zap.Logger, APPconn net.Conn) {
	for {
		data := make([]byte, 1024)
		n, err := APPconn.Read(data)
		if err == io.EOF {
			logger.Error("APP connection closed")
			os.Exit(1)
			return
		} else if err != nil {
			logger.Error("error reading from APP", zap.Error(err))
			return
		}

		//logger.Info(hex.EncodeToString(data[:n]), zap.String("dir", "RX"), zap.Int("n", n))

		connMap.Range(func(key, value interface{}) bool {
			if conn, ok := value.(net.Conn); ok {
				if _, err := conn.Write(data[:n]); err != nil {
					logger.Error("error on writing to connection", zap.Error(err))
				}
			}
			return true
		})
	}
}

func appTX(id string, conn net.Conn, connMap *sync.Map, logger *zap.Logger, APPconn net.Conn) {
	defer func() {
		conn.Close()
		connMap.Delete(id)
	}()

	for {
		data := make([]byte, 1024)
		n, err := conn.Read(data)
		if err == io.EOF {
			logger.Info("connection closed", zap.String("id", id), zap.String("addr", conn.RemoteAddr().String()))
			return
		} else if err != nil {
			logger.Error("error reading from client", zap.Error(err))
			return
		}
		//logger.Info(hex.EncodeToString(data[:n]), zap.String("dir", "TX"), zap.Int("n", n))

		_, err = APPconn.Write(data[:n])
		if err != nil {
			logger.Error("error writing to APP", zap.Error(err))
			return
		}
	}
}
