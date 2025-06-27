package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

const (
	PORT = "12345"
	HOST = "0.0.0.0"
)

var (
	// As variáveis relacionadas a mutex, latencies, failures foram do tcp_client.go
	// e não pertencem ao server.go. Elas não estavam aqui originalmente, então não as adicionei.
	// Se elas estavam aqui por engano, elas já não estão.
)

func handleClient(conn net.Conn, addr string, podName string) {
	defer conn.Close()
	fmt.Printf("[+] Conectado por %s\n", addr)

	buf := make([]byte, 1024)

	for {
		n, err := conn.Read(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("[!] Erro com %s: %v\n", addr, err)
			break
		}

		data := buf[:n]
		message := string(data)
		fmt.Printf("[%s] Recebido: %s\n", addr, message)

		_, err = conn.Write(data)
		if err != nil {
			fmt.Printf("[!] Erro ao enviar para %s: %v\n", addr, err)
			break
		}
		//break // apenas uma mensagem por conexão - Mantenha esta linha comentada ou remova conforme a lógica desejada
	}

	fmt.Printf("[-] Conexão encerrada com %s\n", addr)
}

func main() {
	flag.Parse() // Ainda precisa chamar flag.Parse() mesmo sem flags definidas, caso o Go tenha flags internas.

	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podName = "unknown"
	}

	address := net.JoinHostPort(HOST, PORT)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Printf("[!] Erro ao iniciar o servidor: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("[%s] Servidor escutando na porta %s\n", podName, PORT)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("[!] Erro ao aceitar conexão: %v\n", err)
			continue
		}

		addr := conn.RemoteAddr().String()
		fmt.Printf("[%s] Nova conexão de %s\n", podName, addr)

		go handleClient(conn, addr, podName)
	}
}
