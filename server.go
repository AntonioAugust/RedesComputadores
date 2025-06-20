package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	///"strconv"
	"sync"
	"time"
)

const (
	PORT = "12345"
	HOST = "0.0.0.0"
)

var (
	logChannel = make(chan []string, 100)
	latencyChannel = make(chan time.Duration, 100)
	wg          sync.WaitGroup
)

func handleClient(conn net.Conn, addr string, podName string) {
	defer conn.Close()
	fmt.Printf("[+] Conectado por %s\n", addr)

	buf := make([]byte, 1024)
	start := time.Now()

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

		logChannel <- []string{
			time.Now().Format(time.RFC3339),
			addr,
			fmt.Sprintf("%d", len(data)),
			message,
			podName,
		}

		_, err = conn.Write(data)
		if err != nil {
			fmt.Printf("[!] Erro ao enviar para %s: %v\n", addr, err)
			break
		}
		//break // apenas uma mensagem por conexão
	}

	latency := time.Since(start)
	latencyChannel <- latency

	fmt.Printf("[-] Conexão encerrada com %s\n", addr)
}

func startCSVLogger(filePath string) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("[!] Erro ao abrir arquivo CSV: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	info, _ := file.Stat()
	if info.Size() == 0 {
		writer.Write([]string{"timestamp", "client_ip", "size_bytes", "message", "pod_name"})
	}

	for entry := range logChannel {
		writer.Write(entry)
		writer.Flush()
	}
	wg.Done()
}

func startLatencyAggregator(filePath string) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("[!] Erro ao criar arquivo de latência: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"timestamp", "avg_latency_ms"})

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var latencies []time.Duration

	for {
		select {
		case latency := <-latencyChannel:
			latencies = append(latencies, latency)
		case <-ticker.C:
			if len(latencies) > 0 {
				total := time.Duration(0)
				for _, l := range latencies {
					total += l
				}
				avg := total / time.Duration(len(latencies))
				timestamp := time.Now().Format(time.RFC3339)
				writer.Write([]string{timestamp, fmt.Sprintf("%.2f", avg.Seconds()*1000)})
				writer.Flush()
				latencies = nil
			}
		}
	}
}

func main() {
	logPath := flag.String("log", "logs.csv", "Caminho para o arquivo de log CSV")
	latencyPath := flag.String("latency", "latency_server.csv", "Caminho para o arquivo de latência média")
	flag.Parse()

	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podName = "unknown"
	}

	wg.Add(1)
	go startCSVLogger(*logPath)
	go startLatencyAggregator(*latencyPath)

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

