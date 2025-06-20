#!/bin/bash

set -e

# Variáveis (devem ser as mesmas do seu script principal)
CLUSTER_NAME="tcp-cluster"
IMAGE_NAME="tcp-server:latest" # Nome da imagem Docker que foi criada

echo "--- [✓] Iniciando processo de limpeza do cluster Minikube ---"

# 1. Parar e deletar o cluster Minikube
echo "[!] Deletando cluster Minikube \"$CLUSTER_NAME\"..."
# O comando 'minikube delete' remove o cluster completamente, incluindo volumes e caches.
minikube delete --profile="$CLUSTER_NAME" || { echo "Aviso: Falha ao deletar o cluster Minikube. Pode não existir ou haver outro problema." ; }

echo "[✓] Cluster removido ou não existente."

# 2. Limpar a imagem Docker localmente (opcional, mas recomendado)
# Isso remove a imagem que foi buildada no ambiente Docker do Minikube.
echo "[!] Removendo imagem Docker local \"$IMAGE_NAME\"..."
docker rmi "$IMAGE_NAME" || { echo "Aviso: Falha ao remover a imagem Docker '$IMAGE_NAME'. Pode não existir." ; }
echo "[✓] Imagem Docker removida ou não existente."

# 3. Remover manifestos Kubernetes gerados dinamicamente
if [[ -f "tcp_server.yaml" ]]; then
  echo "[!] Removendo arquivo tcp_server.yaml..."
  rm tcp_server.yaml
  echo "[✓] Arquivo tcp_server.yaml removido."
else
  echo "[ ] Arquivo tcp_server.yaml não encontrado para remoção (já limpo ou não gerado)."
fi

# 4. Remover arquivos CSV de resultados e gráficos gerados (se existirem)
echo "[!] Removendo arquivos de resultados (.csv e .png)..."
find . -maxdepth 1 -type f -name "*.csv" -delete || true
find . -maxdepth 1 -type f -name "*.png" -delete || true
echo "[✓] Arquivos de resultados removidos ou não encontrados."


echo "--- [✓] Limpeza de recursos finalizada! ---"
