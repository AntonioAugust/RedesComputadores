import pandas as pd
import matplotlib.pyplot as plt
import argparse
import os
import itertools # Para criar combinações de parâmetros para os plots

def plot_results(csv_file):
    """
    Lê o arquivo CSV de resultados de teste de carga consolidado e gera gráficos comparativos.
    """
    if not os.path.exists(csv_file):
        print(f"Erro: O arquivo CSV '{csv_file}' não foi encontrado.")
        return

    try:
        df = pd.read_csv(csv_file)
    except Exception as e:
        print(f"Erro ao ler o arquivo CSV '{csv_file}': {e}")
        return

    # Garante que as colunas numéricas são do tipo correto
    # As novas colunas são 'Servidores' e 'MensagensPorCliente'
    df['Servidores'] = pd.to_numeric(df['Servidores'])
    df['Clientes'] = pd.to_numeric(df['Clientes'])
    df['MensagensPorCliente'] = pd.to_numeric(df['MensagensPorCliente'])
    df['LatenciaMedia(ms)'] = pd.to_numeric(df['LatenciaMedia(ms)'])
    df['Sucessos'] = pd.to_numeric(df['Sucessos'])
    df['Falhas'] = pd.to_numeric(df['Falhas'])

    print(f"--- Dados lidos de '{csv_file}': ---")
    print(df.head()) # Mostra as primeiras linhas para verificar
    print(f"Total de registros: {len(df)}")
    print("-------------------------------------")

    # Obter valores únicos para iterar
    unique_servers = sorted(df['Servidores'].unique())
    unique_messages = sorted(df['MensagensPorCliente'].unique())
    unique_clients = sorted(df['Clientes'].unique())

    # --- Gráfico 1: Latência Média vs. Clientes, separando por Servidores e Mensagens ---
    print("Gerando gráficos de Latência Média vs. Clientes...")
    # Cria uma figura para cada valor de 'MensagensPorCliente'
    for messages_val in unique_messages:
        plt.figure(figsize=(14, 8))
        plt.title(f'Latência Média vs. Clientes ({messages_val} Mensagens/Cliente)')
        plt.xlabel('Número de Clientes Simultâneos')
        plt.ylabel('Latência Média (ms)')
        plt.grid(True)
        plt.xticks(unique_clients) # Garante que todos os ticks de clientes sejam exibidos

        # Plota uma linha para cada número de Servidores
        for server_val in unique_servers:
            subset = df[(df['Servidores'] == server_val) & (df['MensagensPorCliente'] == messages_val)]
            if not subset.empty:
                # Ordena para garantir que a linha seja desenhada corretamente
                subset = subset.sort_values(by='Clientes')
                plt.plot(subset['Clientes'], subset['LatenciaMedia(ms)'],
                         marker='o', linestyle='-', label=f'{server_val} Servidores')

        plt.legend(title='Servidores')
        plt.tight_layout()
        plot_path = os.path.join(os.path.dirname(csv_file), f'latency_vs_clients_msg{messages_val}.png')
        plt.savefig(plot_path)
        plt.show()
        print(f"Gráfico salvo em: {plot_path}")

    # --- Gráfico 2: Latência Média vs. Servidores, separando por Clientes e Mensagens ---
    print("\nGerando gráficos de Latência Média vs. Servidores...")
    # Cria uma figura para cada valor de 'MensagensPorCliente'
    for messages_val in unique_messages:
        plt.figure(figsize=(14, 8))
        plt.title(f'Latência Média vs. Servidores ({messages_val} Mensagens/Cliente)')
        plt.xlabel('Número de Servidores')
        plt.ylabel('Latência Média (ms)')
        plt.grid(True)
        plt.xticks(unique_servers) # Garante que todos os ticks de servidores sejam exibidos

        # Plota uma linha para cada número de Clientes
        for client_val in unique_clients:
            subset = df[(df['Clientes'] == client_val) & (df['MensagensPorCliente'] == messages_val)]
            if not subset.empty:
                subset = subset.sort_values(by='Servidores')
                plt.plot(subset['Servidores'], subset['LatenciaMedia(ms)'],
                         marker='o', linestyle='-', label=f'{client_val} Clientes')

        plt.legend(title='Clientes')
        plt.tight_layout()
        plot_path = os.path.join(os.path.dirname(csv_file), f'latency_vs_servers_msg{messages_val}.png')
        plt.savefig(plot_path)
        plt.show()
        print(f"Gráfico salvo em: {plot_path}")

    # --- Gráfico 3: Sucessos e Falhas por Clientes e Servidores (Subplots ou Múltiplos Gráficos) ---
    print("\nGerando gráficos de Sucessos e Falhas...")
    # Podemos criar um grid de subplots para visualizar Sucessos/Falhas
    # Iterar sobre combinações de Servidores e MensagensPorCliente
    for messages_val in unique_messages:
        for server_val in unique_servers:
            subset = df[(df['Servidores'] == server_val) & (df['MensagensPorCliente'] == messages_val)]
            if not subset.empty:
                subset = subset.sort_values(by='Clientes')
                plt.figure(figsize=(12, 7))
                width = 0.35 # Largura das barras

                # Plotar barras de sucessos e falhas
                plt.bar(subset['Clientes'] - width/2, subset['Sucessos'], width, label='Sucessos', color='lightgreen')
                plt.bar(subset['Clientes'] + width/2, subset['Falhas'], width, label='Falhas', color='salmon')

                plt.title(f'Sucessos e Falhas ({server_val} Servidores, {messages_val} Mensagens/Cliente)')
                plt.xlabel('Número de Clientes Simultâneos')
                plt.ylabel('Contagem')
                plt.xticks(subset['Clientes']) # Exibir todos os ticks de clientes testados
                plt.legend()
                plt.grid(axis='y')
                plt.tight_layout()
                plot_path = os.path.join(os.path.dirname(csv_file), f'success_failure_s{server_val}_m{messages_val}.png')
                plt.savefig(plot_path)
                plt.show()
                print(f"Gráfico salvo em: {plot_path}")

    print("\n--- Geração de gráficos concluída! ---")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Gera gráficos a partir do CSV de resultados de teste de carga.")
    parser.add_argument("--csv", type=str, default="full_load_test_results.csv",
                        help="Caminho para o arquivo CSV consolidado gerado pelo cliente Go.")
    args = parser.parse_args()

    # Verifica se as bibliotecas necessárias estão instaladas
    try:
        import pandas
        import matplotlib.pyplot
    except ImportError:
        print("Erro: As bibliotecas 'pandas' e 'matplotlib' não estão instaladas.")
        print("Por favor, instale-as usando: pip install pandas matplotlib")
        exit(1)

    plot_results(args.csv)
