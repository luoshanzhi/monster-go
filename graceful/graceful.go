package graceful

import (
	"context"
	"github.com/luoshanzhi/monster-go"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

func SignalHandler(files []File, callback func(ctx context.Context)) {
	ch := make(chan os.Signal, 1)
	//SIGHUP: hong up 挂断。本信号在用户终端连接(正常或非正常)结束时发出, 通常是在终端的控制进程结束时, 通知同一session内的各个作业, 这时它们与控制终端不再关联。登录Linux时，系统会分配给登录用户一个终端(Session)。在这个终端运行的所有程序，包括前台进程组和 后台进程组，一般都属于这个 Session。当用户退出Linux登录时，前台进程组和后台有对终端输出的进程将会收到SIGHUP信号。这个信号的默认操作为终止进程，因此前台进 程组和后台有终端输出的进程就会中止。不过可以捕获这个信号，比如wget能捕获SIGHUP信号，并忽略它，这样就算退出了Linux登录，wget也能继续下载。此外，对于与终端脱离关系的守护进程，这个信号用于通知它重新读取配置文件。
	//SIGINT: 程序终止(interrupt)信号, 在用户键入INTR字符(通常是Ctrl-C)时发出，用于通知前台进程组终止进程。
	//SIGQUIT: 和SIGINT类似, 但由QUIT字符(通常是Ctrl-\)来控制. 进程在因收到SIGQUIT退出时会产生core文件, 在这个意义上类似于一个程序错误信号。
	//SIGTERM: 程序结束(terminate)信号, 与SIGKILL不同的是该信号可以被阻塞和处理。通常用来要求程序自己正常退出，shell命令kill缺省产生这个信号。如果进程终止不了，我们才会尝试SIGKILL。
	//SIGKILL: 用来立即结: 程序的运行. 本信号不能被阻塞、处理和忽略。如果管理员发现某个进程终止不了，可尝试发送这个信号。
	//SIGSTOP: 停止(stopped)进程的执行. 注意它和terminate以及interrupt的区别:该进程还未结束, 只是暂停执行. 本信号不能被阻塞, 处理或忽略.
	//kill pid, kill -15 pid, kill -SIGTERM pid
	//系统会发送一个SIGTERM的信号给对应的程序。当程序接收到该signal后，将会发生以下的事情
	//程序立刻停止
	//当程序释放相应资源后再停止
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		// timeout context for shutdown
		//ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
		ctx := context.Background()
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			signal.Stop(ch)
			callback(ctx)
			monster.CommonLog.Info("程序退出成功")
			return
		case syscall.SIGUSR2:
			err := reload(files)
			callback(ctx)
			if err != nil {
				monster.CommonLog.Info("程序热更新异常", err)
			} else {
				monster.CommonLog.Info("程序热更新成功")
			}
			return
		}
	}
}

func reload(files []File) error {
	var graceful []string
	var extraFiles []*os.File
	for _, item := range files {
		graceful = append(graceful, strings.TrimSpace(item.Addr))
		extraFiles = append(extraFiles, item.File)
	}
	args := []string{"-graceful", strings.Join(graceful, ",")}
	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// 把 socket FD(文件描述符) ExtraFiles
	cmd.ExtraFiles = extraFiles
	//Run: 执行命令并等待命令执行结束
	//Start: 执行命令但不等待执行结果
	return cmd.Start()
}
