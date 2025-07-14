# RedesComputadores

Testes de carga com várias instâncias simultâneas de um mesmo servidor echo em pods diferentes do cluster.

**Estrutura Básica**:
- **server.go e server.py**: Código de implementação dos servidores.
- **tcp_client.go e tcp_cliente.py**: Código de implementação dos clientes.
- **Dockerfile**: Dockefile para criação da imagem do servidor no cluster.
- **visualizar.py**: Código para criar gráficos e visualizar os dados de métrica.
- **setup_cluster.sh**: script para iniciar o teste de desempenho.
- **delete_cluster.sh**: script para remover o cluster e eliminar resquícios dos testes.
- **tcp_server.yaml**: arquivo de configuração dos pod's do cluster.

**Experimentos realizados no sitema operacional Linux(Ubuntu 22.04)**

**Configuração do Ambiente**:
- Necessário instalar o docker;(https://download.docker.com/linux/ubuntu/gpg)
- Necessaŕio isntalar o minikube(https://minikube-sigs-k8s-io.translate.goog/docs/start/?_x_tr_sl=en&_x_tr_tl=pt&_x_tr_hl=pt&_x_tr_pto=tc&arch=%2Fwindows%2Fx86-64%2Fstable%2F.exe+download);
- Necessário instalar o python;(sudo apt install python3 python3-pip)
- Necessário instalar o go;
- Bibliotecas necessárias:
- pandas(pip install pandas), matplotlib(pip install matplotlib), scipy(pip install scipy).
  
Como usar scripts:

./setup_cluster.sh --replicas "2 4 6 8 10" --messages "1 10 100 500 1000 10000 " --runs 10 

./stop.sh 





