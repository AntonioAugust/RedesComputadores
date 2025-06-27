#!/bin/bash

set -e # Sai imediatamente se um comando falhar

# --- Variáveis de Configuração Padrão ---
CLUSTER_NAME="tcp-cluster"
IMAGE_NAME="tcp-server:latest"
OUTPUT_GLOBAL_CSV="full_load_test_results.csv"
PYTHON_VISUALIZER_SCRIPT="visualizar.py"

# Valores padrão para os parâmetros do teste
# Eles serão sobrescritos se argumentos forem passados pela linha de comando
DEFAULT_REPLICAS="2 4 6 8 10"
DEFAULT_MESSAGES="1 10 100"
DEFAULT_RUNS=10

# Variáveis que armazenarão os valores finais (padrão ou dos argumentos)
REPLICAS_ARRAY=()
MESSAGES_PER_CLIENT_ARRAY=()
NUMBER_OF_RUNS=0

# --- Funções ---

# Função de limpeza para garantir que os recursos sejam removidos
cleanup() {
  echo "--- [✓] Executando limpeza de recursos ---"
  kubectl delete -f tcp_server.yaml --ignore-not-found || true
  echo "--- Limpeza concluída! ---"
}

# Registra a função de limpeza para ser executada na saída do script (mesmo em caso de erro)
trap cleanup EXIT

# --- Parse de Argumentos ---
# Loop para processar os argumentos passados
while [[ $# -gt 0 ]]; do
  case "$1" in
    --replicas)
      # Pega o próximo argumento (que é a string de réplicas)
      REPLICAS_ARRAY=($2) # Converte a string em array, ex: "2 4 6" vira (2 4 6)
      shift 2 # Move para o próximo par de argumento/valor
      ;;
    --messages)
      MESSAGES_PER_CLIENT_ARRAY=($2)
      shift 2
      ;;
    --runs)
      if ! [[ "$2" =~ ^[0-9]+$ ]]; then
        echo "Erro: O valor para --runs deve ser um número inteiro."
        exit 1
      fi
      NUMBER_OF_RUNS="$2"
      shift 2
      ;;
    --output-csv)
      OUTPUT_GLOBAL_CSV="$2"
      shift 2
      ;;
    *)
      # Caso um argumento desconhecido seja passado
      echo "Uso: $0 [--replicas \"N1 N2...\"] [--messages \"M1 M2...\"] [--runs N] [--output-csv ARQUIVO.csv]"
      echo "Exemplo: $0 --replicas \"2 4\" --messages \"10 100\" --runs 5"
      exit 1
      ;;
  esac
done

# --- Definir valores padrão se nenhum argumento foi fornecido ---
# Se o array de réplicas estiver vazio, usa o padrão
if [ ${#REPLICAS_ARRAY[@]} -eq 0 ]; then
  REPLICAS_ARRAY=($DEFAULT_REPLICAS)
fi
# Se o array de mensagens estiver vazio, usa o padrão
if [ ${#MESSAGES_PER_CLIENT_ARRAY[@]} -eq 0 ]; then
  MESSAGES_PER_CLIENT_ARRAY=($DEFAULT_MESSAGES)
fi
# Se o número de rodadas for zero (não definido), usa o padrão
if [ "$NUMBER_OF_RUNS" -eq 0 ]; then
  NUMBER_OF_RUNS="$DEFAULT_RUNS"
fi

# --- Início da Execução Principal ---

echo "--- [✓] Iniciando Minikube ($CLUSTER_NAME) ---"
minikube start --profile="$CLUSTER_NAME"

echo "--- [✓] Configurando ambiente Docker do Minikube ---"
if ! eval "$(minikube -p "$CLUSTER_NAME" docker-env)"; then
  echo "Erro: Não foi possível configurar o ambiente Docker do Minikube. O Docker está rodando?"
  exit 1
fi

echo "--- [✓] Buildando a imagem Docker ($IMAGE_NAME) ---"
docker build -t "$IMAGE_NAME" .

# --- Preparação do Arquivo CSV Global de Resultados (UMA ÚNICA VEZ) ---
echo "--- [✓] Limpando e preparando o arquivo de resultados global: $OUTPUT_GLOBAL_CSV ---"
rm -f "$OUTPUT_GLOBAL_CSV"
# Certifique-se de que este cabeçalho corresponde exatamente às colunas do tcp_client.go
echo "Rodada,Servidores,Clientes,MensagensPorCliente,LatenciaMedia(ms),LatenciaMin(ms),LatenciaMax(ms),LatenciaMediana(ms),LatenciaStdDev(ms),Sucessos,Falhas" > "$OUTPUT_GLOBAL_CSV"

# --- Loop MAIS EXTERNO para o Número de Rodadas Completas ---
for run_id in $(seq 1 "$NUMBER_OF_RUNS"); do
  echo "========================================================================="
  echo "=============== INICIANDO RODADA COMPLETA NÚMERO: $run_id de $NUMBER_OF_RUNS ==============="
  echo "========================================================================="

  # --- Loops Aninhados para Cenários de Teste ---

  # Loop para o número de réplicas (servidores)
  for current_replicas in "${REPLICAS_ARRAY[@]}"; do
    echo "--- [✓] Rodada $run_id: Configurando Deployment com $current_replicas réplicas ---"

    cat <<EOF > tcp_server.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tcp-server-deployment
spec:
  replicas: $current_replicas
  selector:
    matchLabels:
      app: tcp-server
  template:
    metadata:
      labels:
        app: tcp-server
    spec:
      containers:
      - name: tcp-server
        image: $IMAGE_NAME
        imagePullPolicy: Never
        ports:
        - containerPort: 12345
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
---
apiVersion: v1
kind: Service
metadata:
  name: tcp-server-service
spec:
  selector:
    app: tcp-server
  ports:
    - protocol: TCP
      port: 12345
      targetPort: 12345
  type: NodePort
EOF

    echo "--- [✓] Rodada $run_id: Aplicando Deployment e Service para $current_replicas réplicas ---"
    kubectl apply -f tcp_server.yaml

    echo "--- [✓] Rodada $run_id: Aguardando pods de $current_replicas réplicas ficarem prontos ---"
    kubectl rollout status deployment/tcp-server-deployment --timeout=300s

    echo "--- [✓] Rodada $run_id: Obtendo IP e porta do serviço TCP ---"
    SERVER_IP=$(minikube ip -p "$CLUSTER_NAME")
    if [ -z "$SERVER_IP" ]; then
      echo "Erro: Não foi possível obter o IP do Minikube."
      exit 1
    fi

    SERVER_PORT=$(kubectl get svc tcp-server-service -o jsonpath='{.spec.ports[0].nodePort}')
    if [ -z "$SERVER_PORT" ]; then
      echo "Erro: Não foi possível obter a porta do serviço TCP."
      exit 1
    fi

    # Loop para o número de mensagens por cliente
    for current_messages_per_client in "${MESSAGES_PER_CLIENT_ARRAY[@]}"; do
      echo "--- [✓] Rodada $run_id: Iniciando testes para $current_replicas réplicas, $current_messages_per_client mensagens por cliente ---"

      go run tcp_client.go \
        --ip "$SERVER_IP" \
        --port "$SERVER_PORT" \
        --messages "$current_messages_per_client" \
        --output "$OUTPUT_GLOBAL_CSV" \
        --current-replicas "$current_replicas" \
        --current-test-messages "$current_messages_per_client" \
        --run-id "$run_id"

      echo "--- [✓] Rodada $run_id: Testes para esta combinação concluídos. Pausando por 5 segundos ---"
      sleep 5
    done

    echo "--- [✓] Rodada $run_id: Limpando Deployment e Service atuais para $current_replicas réplicas ---"
    kubectl delete -f tcp_server.yaml --ignore-not-found || true
    sleep 2
  done
  echo "--- [✓] Rodada $run_id completa. Próxima rodada em 10 segundos... ---"
  sleep 10
done

echo "--- [✓] Todas as $NUMBER_OF_RUNS rodadas de testes de carga concluídas! ---"

# --- Visualização dos Resultados ---
echo "--- [✓] Gerando gráficos de latência e falhas a partir de $OUTPUT_GLOBAL_CSV ---"
python3 "$PYTHON_VISUALIZER_SCRIPT" --csv "$OUTPUT_GLOBAL_CSV"

echo "--- [✓] Processo completo finalizado! ---"
