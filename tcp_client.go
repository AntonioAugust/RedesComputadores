package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	message           = []byte("teste de desempenho")
	connectionTimeout = 5 * time.Second
	mutex             sync.Mutex
	latencies         []float64
	failures          int
)

func connectAndSend(index int, serverIP string, serverPort int, messagesPerClient int, wg *sync.WaitGroup) {
	defer wg.Done()

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", serverIP, serverPort), connectionTimeout)
	if err != nil {
		mutex.Lock()
		failures++
		mutex.Unlock()
		fmt.Printf("[!] Cliente %d falhou: %v\n", index, err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for i := 0; i < messagesPerClient; i++ {
		_, err := conn.Write(message)
		if err != nil {
			mutex.Lock()
			failures++
			mutex.Unlock()
			fmt.Printf("[!] Cliente %d erro ao enviar: %v\n", index, err)
			return
		}
		time.Sleep(10 * time.Millisecond) // Considerar remover ou ajustar se a intenção é medir a latência pura
		conn.SetReadDeadline(time.Now().Add(connectionTimeout))
		_, err = conn.Read(buf)
		if err != nil {
			mutex.Lock()
			failures++
			mutex.Unlock()
			fmt.Printf("[!] Cliente %d erro ao receber: %v\n", index, err)
			return
		}
	}
	end := time.Now()

	mutex.Lock()
	latencies = append(latencies, end.Sub(start).Seconds())
	mutex.Unlock()
}

// runTest agora recebe as réplicas e mensagens por cliente para inclusão no CSV
func runTest(
	clientCount int,
	serverIP string,
	serverPort int,
	messagesPerClientParam int, // Renomeado para evitar conflito com a flag de entrada
	currentReplicas int,        // Novo parâmetro: número de réplicas nesta execução
	messagesPerClientTest int,  // Novo parâmetro: número de mensagens por cliente nesta execução de teste
	writer *csv.Writer,
) {
	latencies = nil
	failures = 0

	fmt.Printf("\n=== Teste: Réplicas=%d, Mensagens por Cliente=%d, Clientes Simultâneos=%d ===\n",
		currentReplicas, messagesPerClientTest, clientCount)

	var wg sync.WaitGroup
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		// Passa 'messagesPerClientTest' para a função 'connectAndSend'
		go connectAndSend(i, serverIP, serverPort, messagesPerClientTest, &wg)
	}
	wg.Wait()

	if len(latencies) > 0 {
		var total float64
		for _, lat := range latencies {
			total += lat
		}
		avg := total / float64(len(latencies))
		fmt.Printf("[✓] Latência média por cliente: %.2f ms\n", avg*1000)
		fmt.Printf("[✓] Sucesso: %d | Falhas: %d\n", len(latencies), failures)
		writer.Write([]string{
			strconv.Itoa(currentReplicas),       // Nova coluna: Servidores (Réplicas)
			strconv.Itoa(clientCount),           // Clientes
			strconv.Itoa(messagesPerClientTest), // Nova coluna: Mensagens por Cliente para ESTA execução
			fmt.Sprintf("%.2f", avg*1000),       // LatenciaMedia(ms)
			strconv.Itoa(len(latencies)),        // Sucessos
			strconv.Itoa(failures),              // Falhas
		})
	} else {
		fmt.Println("[✗] Nenhuma conexão bem-sucedida.")
		writer.Write([]string{
			strconv.Itoa(currentReplicas),       // Nova coluna: Servidores (Réplicas)
			strconv.Itoa(clientCount),           // Clientes
			strconv.Itoa(messagesPerClientTest), // Nova coluna: Mensagens por Cliente para ESTA execução
			"0", "0", strconv.Itoa(clientCount), // LatenciaMedia(ms), Sucessos, Falhas (todos zero se não houver sucessos)
		})
	}
	writer.Flush()
}

func main() {
	// Flags de linha de comando
	ip := flag.String("ip", "127.0.0.1", "Endereço IP do servidor")
	port := flag.Int("port", 12345, "Porta do servidor")
	// Este 'messages' agora representa o número de mensagens POR CLIENTE para a execução ATUAL
	messages := flag.Int("messages", 1, "Número de mensagens por cliente para cada cliente neste teste.")
	output := flag.String("output", "full_load_test_results.csv", "Arquivo CSV de saída para todos os resultados")
	// Novas flags para os parâmetros externos (réplicas e mensagens da iteração atual do script shell)
	currentReplicas := flag.Int("current-replicas", 0, "Número de réplicas do servidor para esta execução (para registro no CSV)")
	currentTestMessages := flag.Int("current-test-messages", 0, "Número de mensagens por cliente para o teste atual (para registro no CSV)")
	flag.Parse()

	// Abre o arquivo CSV em modo de apêndice. Se não existir, ele será criado.
	file, err := os.OpenFile(*output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Erro ao abrir/criar arquivo CSV:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	// Verifica se o arquivo está vazio para escrever o cabeçalho
	// (isto é uma heurística, idealmente o script shell seria responsável pelo cabeçalho único)
	stat, _ := file.Stat()
	if stat.Size() == 0 {
		writer.Write([]string{"Servidores", "Clientes", "MensagensPorCliente", "LatenciaMedia(ms)", "Sucessos", "Falhas"})
		writer.Flush()
	}

	// Loop de clientes (10 a 100) que o seu cliente Go já faz
	for n := 10; n <= 100; n += 10 {
		// Passamos os parâmetros da execução atual (currentReplicas e currentTestMessages)
		// para serem registrados no CSV, além dos parâmetros internos do loop (n=clientCount).
		runTest(n, *ip, *port, *messages, *currentReplicas, *currentTestMessages, writer)
		time.Sleep(2 * time.Second) // Pequena pausa entre cada conjunto de clientes
	}
}
