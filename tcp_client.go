package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"net"
	"os"
	"sort" // Necessário para calcular a mediana
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
		// Omitindo a impressão de erro para cada falha de conexão individual para evitar muita saída no console
		// fmt.Printf("[!] Cliente %d falhou: %v\n", index, err)
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
			// Omitindo a impressão de erro para cada falha de envio individual
			// fmt.Printf("[!] Cliente %d erro ao enviar: %v\n", index, err)
			return
		}
		time.Sleep(10 * time.Millisecond) // Considerar remover ou ajustar se a intenção é medir a latência pura
		conn.SetReadDeadline(time.Now().Add(connectionTimeout))
		_, err = conn.Read(buf)
		if err != nil {
			mutex.Lock()
			failures++
			mutex.Unlock()
			// Omitindo a impressão de erro para cada falha de recebimento individual
			// fmt.Printf("[!] Cliente %d erro ao receber: %v\n", index, err)
			return
		}
	}
	end := time.Now()

	mutex.Lock()
	latencies = append(latencies, end.Sub(start).Seconds())
	mutex.Unlock()
}

// runTest agora recebe as réplicas, mensagens por cliente e o ID da rodada para inclusão no CSV
func runTest(
	clientCount int,
	serverIP string,
	serverPort int,
	messagesPerClientParam int, // Valor da flag --messages
	currentReplicas int,        // Parâmetro do script shell: número de réplicas
	messagesPerClientTest int,  // Parâmetro do script shell: número de mensagens por cliente para o teste atual
	runID int,                  // NOVO PARÂMETRO: ID da rodada atual
	writer *csv.Writer,
) {
	latencies = nil // Resetar latências para esta execução específica de runTest
	failures = 0    // Resetar falhas para esta execução específica de runTest

	fmt.Printf("\n=== Rodada %d: Réplicas=%d, Mensagens por Cliente=%d, Clientes Simultâneos=%d ===\n",
		runID, currentReplicas, messagesPerClientTest, clientCount)

	var wg sync.WaitGroup
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go connectAndSend(i, serverIP, serverPort, messagesPerClientTest, &wg)
	}
	wg.Wait()

	// Cálculo das estatísticas para esta *única* execução de runTest
	var avgLatencyMs float64
	var minLatencyMs float64
	var maxLatencyMs float64
	var medianLatencyMs float64
	var stdDevLatencyMs float64

	if len(latencies) > 0 {
		var total float64
		for _, lat := range latencies {
			total += lat
		}
		avgLatencyMs = (total / float64(len(latencies))) * 1000

		// Converter latências para ms para cálculo de min/max/median/stddev
		latenciesMs := make([]float64, len(latencies))
		for i, l := range latencies {
			latenciesMs[i] = l * 1000
		}

		// Min
		minLatencyMs = latenciesMs[0]
		for _, l := range latenciesMs {
			if l < minLatencyMs {
				minLatencyMs = l
			}
		}

		// Max
		maxLatencyMs = latenciesMs[0]
		for _, l := range latenciesMs {
			if l > maxLatencyMs {
				maxLatencyMs = l
			}
		}

		// Mediana
		sort.Float64s(latenciesMs)
		mid := len(latenciesMs) / 2
		if len(latenciesMs)%2 == 0 {
			medianLatencyMs = (latenciesMs[mid-1] + latenciesMs[mid]) / 2
		} else {
			medianLatencyMs = latenciesMs[mid]
		}

		// Desvio Padrão
		var sumSquares float64
		for _, l := range latenciesMs {
			diff := l - avgLatencyMs
			sumSquares += diff * diff
		}
		stdDevLatencyMs = 0.0
		if len(latenciesMs) > 1 {
			stdDevLatencyMs = (sumSquares / float64(len(latenciesMs)-1)) // Amostral
			// stdDevLatencyMs = (sumSquares / float64(len(latenciesMs))) // População
			stdDevLatencyMs = stdDevLatencyMs * stdDevLatencyMs // sqrt of variance to get std dev
		}

		fmt.Printf("[✓] Latência média por cliente: %.2f ms\n", avgLatencyMs)
		fmt.Printf("[✓] Latência mínima: %.2f ms, Máxima: %.2f ms, Mediana: %.2f ms, Desvio Padrão: %.2f ms\n",
			minLatencyMs, maxLatencyMs, medianLatencyMs, stdDevLatencyMs)
		fmt.Printf("[✓] Sucesso: %d | Falhas: %d\n", len(latencies), failures)

		// Escreve a linha no CSV
		writer.Write([]string{
			strconv.Itoa(runID),                       // Nova coluna: Rodada
			strconv.Itoa(currentReplicas),             // Servidores (Réplicas)
			strconv.Itoa(clientCount),                 // Clientes
			strconv.Itoa(messagesPerClientTest),       // Mensagens por Cliente para ESTA execução
			fmt.Sprintf("%.2f", avgLatencyMs),         // LatenciaMedia(ms)
			fmt.Sprintf("%.2f", minLatencyMs),         // LatenciaMin(ms)
			fmt.Sprintf("%.2f", maxLatencyMs),         // LatenciaMax(ms)
			fmt.Sprintf("%.2f", medianLatencyMs),      // LatenciaMediana(ms)
			fmt.Sprintf("%.2f", stdDevLatencyMs),      // LatenciaStdDev(ms)
			strconv.Itoa(len(latencies)),              // Sucessos
			strconv.Itoa(failures),                    // Falhas
		})
	} else {
		fmt.Println("[✗] Nenhuma conexão bem-sucedida.")
		writer.Write([]string{
			strconv.Itoa(runID),                       // Nova coluna: Rodada
			strconv.Itoa(currentReplicas),             // Servidores (Réplicas)
			strconv.Itoa(clientCount),                 // Clientes
			strconv.Itoa(messagesPerClientTest),       // Mensagens por Cliente para ESTA execução
			"0", "0", "0", "0", "0",                   // Latência média, min, max, mediana, desvio padrão (todos zero)
			"0", strconv.Itoa(clientCount),            // Sucessos (0), Falhas (total de clientes)
		})
	}
	writer.Flush()
}

func main() {
	// Flags de linha de comando
	ip := flag.String("ip", "127.0.0.1", "Endereço IP do servidor")
	port := flag.Int("port", 12345, "Porta do servidor")
	messages := flag.Int("messages", 1, "Número de mensagens por cliente para cada cliente neste teste.")
	output := flag.String("output", "full_load_test_results.csv", "Arquivo CSV de saída para todos os resultados")
	currentReplicas := flag.Int("current-replicas", 0, "Número de réplicas do servidor para esta execução (para registro no CSV)")
	currentTestMessages := flag.Int("current-test-messages", 0, "Número de mensagens por cliente para o teste atual (para registro no CSV)")
	runID := flag.Int("run-id", 0, "ID da rodada de teste atual (para registro no CSV)") // NOVA FLAG: ID da Rodada
	flag.Parse()

	// Abre o arquivo CSV em modo de apêndice. Se não existir, ele será criado.
	file, err := os.OpenFile(*output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Erro ao abrir/criar arquivo CSV:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	// Loop de clientes (10 a 100) que o seu cliente Go já faz
	for n := 10; n <= 100; n += 10 {
		// Passamos os parâmetros da execução atual, incluindo o novo runID
		runTest(n, *ip, *port, *messages, *currentReplicas, *currentTestMessages, *runID, writer)
		time.Sleep(2 * time.Second) // Pequena pausa entre cada conjunto de clientes
	}
}
