package info

import (
	"sync"
	"time"

	"github.com/1uvu/nebula-diagnosis-cli/pkg/config"
	"github.com/1uvu/nebula-diagnosis-cli/pkg/logger"
)

func Run(conf *config.InfoConfig) {
	nodeNumber := len(conf.Node)
	wg := sync.WaitGroup{}
	wg.Add(nodeNumber)
	for name := range conf.Node {
		go func(name string) {
			nodeConfig := conf.Node[name]
			_logger := logger.GetLogger(name, nodeConfig.OutputDirPath, nodeConfig.LogToFile)
			// the conf has been verified, so don't need to handle error
			d, _ := time.ParseDuration(nodeConfig.Duration)
			if nodeConfig.Duration == "-1" {
				runWithInfinity(nodeConfig, _logger)
				wg.Done()
			} else if d == 0 {
				run(nodeConfig, _logger)
				wg.Done()
			} else {
				runWithDuration(nodeConfig, _logger)
				wg.Done()
			}
		}(name)
	}
	wg.Wait()
}

func run(nodeConfig *config.NodeConfig, defaultLogger logger.Logger) {
	for _, option := range nodeConfig.Options {
		//fetchAndSaveInfo(nodeConfig, option, defaultLogger)
		defaultLogger.Info(option)
	}
}

func runWithInfinity(nodeConfig *config.NodeConfig, defaultLogger logger.Logger) {
	p, _ := time.ParseDuration(nodeConfig.Period)
	ticker := time.NewTicker(p)
	for {
		select {
		case <-ticker.C:
			run(nodeConfig, defaultLogger)
		default:

		}
	}
}

func runWithDuration(nodeConfig *config.NodeConfig, defaultLogger logger.Logger) {
	p, _ := time.ParseDuration(nodeConfig.Period)
	ticker := time.NewTicker(p)
	ch := make(chan bool)
	go func(ticker *time.Ticker) {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				run(nodeConfig, defaultLogger)
			case stop := <-ch:
				if stop {
					return
				}
			default:

			}
		}
	}(ticker)

	d, _ := time.ParseDuration(nodeConfig.Duration)
	time.Sleep(d)
	ch <- true
	close(ch)
}
