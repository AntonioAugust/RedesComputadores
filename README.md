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

**Rodando no linux**:
Necessário instalar o docker;
Necessário instalar o python;
Necessaŕio instalar o kubernets;
Necessaŕio isntalar o minikube;





