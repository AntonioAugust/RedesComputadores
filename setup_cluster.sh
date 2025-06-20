#!/bin/bash

set -e # Sai imediatamente se um comando falhar

# --- Variáveis de Configuração ---
CLUSTER_NAME="tcp-cluster"
IMAGE_NAME="tcp-server:latest"
OUTPUT_GLOBAL_CSV="full_load_test_results.csv" # Arquivo CSV consolidado para todos os resultados
PYTHON_VISUALIZER_SCRIPT="visualizar.py" # Nome do seu script Python de visualização

# Arrays para as iterações do teste
REPLICAS_ARRAY=(2 4 6 8 10)
MESSAGES_PER_CLIENT_ARRAY=(1 50 100) # Número de mensagens que cada cliente vai enviar

# --- Funções ---

# Função de limpeza para garantir que os recursos sejam removidos
cleanup() {
  echo "--- [✓] Executando limpeza de recursos ---"
  kubectl delete -f tcp_server.yaml --ignore-not-found || true
  # Se você quiser parar o Minikube no final de CADA EXECUÇÃO do script, descomente abaixo.
  # Mas para o loop completo, é melhor parar APENAS no cleanup.sh separado ou no final aqui.
  # minikube stop --profile="$CLUSTER_NAME" || true
  echo "--- Limpeza concluída! ---"
}

# Registra a função de limpeza para ser executada na saída do script (mesmo em caso de erro)
trap cleanup EXIT

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

# --- Preparação do Arquivo CSV Global de Resultados ---
echo "--- [✓] Limpando e preparando o arquivo de resultados global: $OUTPUT_GLOBAL_CSV ---"
# Remove o arquivo CSV global antigo, se existir, para iniciar um novo
rm -f "$OUTPUT_GLOBAL_CSV"
# Escreve o cabeçalho no arquivo CSV global UMA VEZ
echo "Servidores,Clientes,MensagensPorCliente,LatenciaMedia(ms),Sucessos,Falhas" > "$OUTPUT_GLOBAL_CSV"

# --- Loops Aninhados para Cenários de Teste ---

# Loop para o número de réplicas (servidores)
for current_replicas in "${REPLICAS_ARRAY[@]}"; do
  echo "--- [✓] Configurando Deployment com $current_replicas réplicas ---"

  # Gerando o manifesto Kubernetes dinamicamente para o número atual de réplicas
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
        imagePullPolicy: Never # Essencial para usar imagem local do Minikube
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
      port: 12345 # Porta do serviço
      targetPort: 12345 # Porta do container
  type: NodePort # Para fácil acesso do host
EOF

  echo "--- [✓] Aplicando Deployment e Service para $current_replicas réplicas ---"
  kubectl apply -f tcp_server.yaml

  echo "--- [✓] Aguardando pods de $current_replicas réplicas ficarem prontos ---"
  # Adicionado timeout para evitar que o script fique preso
  kubectl rollout status deployment/tcp-server-deployment --timeout=300s

  echo "--- [✓] Obtendo IP e porta do serviço TCP ---"
  SERVER_IP=$(minikube ip -p "$CLUSTER_NAME")
  if [ -z "$SERVER_IP" ]; then
    echo "Erro: Não foi possível obter o IP do Minikube."
    exit 1
  fi
  echo "SERVER IP = $SERVER_IP"

  SERVER_PORT=$(kubectl get svc tcp-server-service -o jsonpath='{.spec.ports[0].nodePort}')
  if [ -z "$SERVER_PORT" ]; then
    echo "Erro: Não foi possível obter a porta do serviço TCP."
    exit 1
  fi
  echo "SERVER PORT = $SERVER_PORT"

  # Loop para o número de mensagens por cliente
  for current_messages_per_client in "${MESSAGES_PER_CLIENT_ARRAY[@]}"; do
    echo "--- [✓] Iniciando testes para $current_replicas réplicas, $current_messages_per_client mensagens por cliente ---"

    # Executa o cliente Go, passando os parâmetros para registro no CSV
    # O cliente Go tem seu próprio loop para 10 a 100 clientes.
    go run tcp_client.go \
      --ip "$SERVER_IP" \
      --port "$SERVER_PORT" \
      --messages "$current_messages_per_client" \
      --output "$OUTPUT_GLOBAL_CSV" \
      --current-replicas "$current_replicas" \
      --current-test-messages "$current_messages_per_client"

    echo "--- [✓] Testes para esta combinação concluídos. Pausando por 5 segundos ---"
    sleep 5 # Pequena pausa entre as diferentes configurações de teste
  done

  # Limpar o deployment e service atuais antes de configurar para a próxima quantidade de réplicas
  echo "--- [✓] Limpando Deployment e Service atuais para $current_replicas réplicas ---"
  kubectl delete -f tcp_server.yaml --ignore-not-found || true
  sleep 2 # Pequena pausa para garantir que os recursos sejam removidos antes da próxima iteração
done

echo "--- [✓] Todos os testes de carga concluídos! ---"

# --- Visualização dos Resultados ---
echo "--- [✓] Gerando gráficos de latência e falhas a partir de $OUTPUT_GLOBAL_CSV ---"
# Certifique-se que o script visualizar.py existe e que as dependências Python estão instaladas
python3 "$PYTHON_VISUALIZER_SCRIPT" --csv "$OUTPUT_GLOBAL_CSV"

echo "--- [✓] Processo completo finalizado! ---"
