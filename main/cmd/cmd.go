package cmd

import (
	"flag"
	"fmt"
	"io/ioutil"
	"github.com/LaYa-op/laya-go/main/logic"
	"github.com/LaYa-op/laya-go/main/utils"
	"os"
	"os/signal"
	"syscall"
)

const (
	AppVersion = "1.0.0"
)

// 命令类
type Cmd struct {
	config utils.Config
	logger utils.Logger
	timer  logic.Timer
	exit   chan bool
}

// 执行
func (p *Cmd) Run() {
	// 命令行参数处理
	daemon, configuration := p.handleFlags()
	// 守护进程
	if daemon {
		utils.Daemon()
	}
	// 变量定义
	p.exit = make(chan bool)
	// 欢迎
	welcome()
	// 实例化公共组件
	p.config = utils.LoadConfig(configuration)
	p.logger = utils.NewLogger(p.config)
	// pid处理
	p.handlePid()
	// 输出启动日志
	p.logger.Info(fmt.Sprintf("Service started successfully, PID: %d", os.Getpid()))
	// 启动定时器
	p.timer = logic.Timer{
		Config: p.config,
		Logger: p.logger,
	}
	p.timer.Init()
	p.timer.Start()
	// 信号处理
	p.handleSignal()
	// 退出
	<-p.exit
	// 输出停止日志
	p.logger.Info(fmt.Sprintf("Service stopped successfully, PID: %d", os.Getpid()))
}

// 欢迎信息
func welcome() {
	fmt.Println("Service:		LaYa-process-Manager")
	fmt.Println("Version:		" + AppVersion)
}

// PID处理
func (p *Cmd) handlePid() {
	// 不处理
	if p.config.Delayer.Pid == "" {
		return
	}
	// 读取
	pidStr, err := ioutil.ReadFile(p.config.Delayer.Pid)
	if err != nil {
		p.writePidFile(p.config.Delayer.Pid)
		return
	}
	// 重复启动处理
	pid, err := utils.ByteToInt(pidStr)
	if err != nil {
		p.writePidFile(p.config.Delayer.Pid)
		return
	}
	pro, err := os.FindProcess(pid)
	if err != nil {
		p.writePidFile(p.config.Delayer.Pid)
		return
	}
	// Win 中全部返回错误: not supported by windows
	err = pro.Signal(os.Signal(syscall.Signal(0)))
	if err != nil {
		// os: process already finished
		// not supported by windows
		p.writePidFile(p.config.Delayer.Pid)
		return
	}
	p.logger.Error(fmt.Sprintf("ERROR: Service is being executed, PID: %d", pid), true)
}

// 写入PID文件
func (p *Cmd) writePidFile(pidFile string) {
	err := ioutil.WriteFile(pidFile, utils.IntToByte(os.Getpid()), 0644)
	if err != nil {
		p.logger.Error(fmt.Sprintf("PID file cannot be written: %s", pidFile), true)
	}
}

// 信号处理
func (p *Cmd) handleSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-ch
		switch sig {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			p.timer.Stop()
			p.exit <- true
		}
	}()
}

// 参数处理
func (p *Cmd) handleFlags() (bool, string) {
	// 参数解析
	flagD := flag.Bool("d", false, "")
	flagDaemon := flag.Bool("daemon", false, "")
	flagH := flag.Bool("h", false, "")
	flagHelp := flag.Bool("help", false, "")
	flagV := flag.Bool("v", false, "")
	flagVersion := flag.Bool("version", false, "")
	flagC := flag.String("c", "", "")
	flagConfiguration := flag.String("configuration", "", "")
	flag.Parse()
	// 参数取值
	daemon := *flagD || *flagDaemon
	help := *flagH || *flagHelp
	version := *flagV || *flagVersion
	configuration := ""
	if *flagC == "" {
		configuration = *flagConfiguration
	} else {
		configuration = *flagC
	}
	// 打印型命令处理
	if help {
		printHelp()
	}
	if version {
		printVersion()
	}
	// 返回参数值
	return daemon, configuration
}

// 打印帮助
func printHelp() {
	fmt.Println("Usage: delayer [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("-d/--daemon run in the background")
	fmt.Println("-c/--configuration FILENAME -- configuration file path (searches if not given)")
	fmt.Println("-h/--help -- print this usage message and exit")
	fmt.Println("-v/--version -- print version number and exit")
	fmt.Println()
	os.Exit(0)
}

// 打印版本
func printVersion() {
	fmt.Println(AppVersion)
	fmt.Println()
	os.Exit(0)
}
