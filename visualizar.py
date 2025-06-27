import pandas as pd
import matplotlib.pyplot as plt
import argparse
import os
import itertools
from scipy import stats # Importa a biblioteca para calcular o Z-score

def plot_results(csv_file, z_score_threshold=3.0): # Adiciona um parâmetro para o threshold
    """
    Lê o arquivo CSV de resultados de teste de carga consolidado,
    identifica e remove outliers usando Z-score, e gera gráficos comparativos.
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
    df['Rodada'] = pd.to_numeric(df['Rodada'])
    df['Servidores'] = pd.to_numeric(df['Servidores'])
    df['Clientes'] = pd.to_numeric(df['Clientes'])
    df['MensagensPorCliente'] = pd.to_numeric(df['MensagensPorCliente'])
    df['LatenciaMedia(ms)'] = pd.to_numeric(df['LatenciaMedia(ms)'])
    df['LatenciaMin(ms)'] = pd.to_numeric(df['LatenciaMin(ms)'])
    df['LatenciaMax(ms)'] = pd.to_numeric(df['LatenciaMax(ms)'])
    df['LatenciaMediana(ms)'] = pd.to_numeric(df['LatenciaMediana(ms)'])
    df['LatenciaStdDev(ms)'] = pd.to_numeric(df['LatenciaStdDev(ms)'])
    df['Sucessos'] = pd.to_numeric(df['Sucessos'])
    df['Falhas'] = pd.to_numeric(df['Falhas'])

    print(f"--- Dados lidos de '{csv_file}': ---")
    print(df.head())
    print(f"Total de registros ANTES da remoção de outliers: {len(df)}")
    print("-------------------------------------")

    # --- Detecção e Remoção de Outliers usando Z-score ---
    print(f"\n--- Aplicando detecção de outliers com Z-score (threshold = ±{z_score_threshold}) ---")

    # Calculamos o Z-score para 'LatenciaMedia(ms)' dentro de CADA GRUPO
    # (Servidores, Clientes, MensagensPorCliente).
    # Isso é crucial porque a média e o desvio padrão de um cenário
    # não devem ser influenciados por outros cenários.
    df['Z_score'] = df.groupby(['Servidores', 'Clientes', 'MensagensPorCliente'])['LatenciaMedia(ms)'].transform(lambda x: stats.zscore(x))

    # Identifica outliers (valores absolutos do Z-score acima do threshold)
    outliers = df[abs(df['Z_score']) > z_score_threshold]

    if not outliers.empty:
        print(f"Outliers detectados ({len(outliers)} registros):")
        print(outliers[['Rodada', 'Servidores', 'Clientes', 'MensagensPorCliente', 'LatenciaMedia(ms)', 'Z_score']])
        print("\nRemovendo outliers...")
        df_cleaned = df[abs(df['Z_score']) <= z_score_threshold].copy() # Cria uma cópia para evitar SettingWithCopyWarning
        print(f"Total de registros DEPOIS da remoção de outliers: {len(df_cleaned)}")
    else:
        print("Nenhum outlier detectado para o threshold especificado.")
        df_cleaned = df.copy() # Se não há outliers, a base limpa é a original

    # Usaremos df_cleaned para todas as análises subsequentes
    df = df_cleaned
    print("---------------------------------------------------------------")


    # Obter valores únicos para iterar
    unique_servers = sorted(df['Servidores'].unique())
    unique_messages = sorted(df['MensagensPorCliente'].unique())
    unique_clients = sorted(df['Clientes'].unique())

    # Group data by the unique test scenarios (excluding 'Rodada')
    # and calculate aggregate statistics across the remaining runs
    grouped_df = df.groupby(['Servidores', 'Clientes', 'MensagensPorCliente']).agg(
        # Statistics for LatenciaMedia(ms) across runs (after outlier removal)
        AvgLatencyAcrossRuns=('LatenciaMedia(ms)', 'mean'),
        MedianLatencyAcrossRuns=('LatenciaMedia(ms)', 'median'),
        MinLatencyAcrossRuns=('LatenciaMedia(ms)', 'min'),
        MaxLatencyAcrossRuns=('LatenciaMedia(ms)', 'max'),
        StdDevLatencyAcrossRuns=('LatenciaMedia(ms)', 'std'),
        # Aggregate Sucessos and Falhas (sum across remaining runs)
        TotalSucessos=('Sucessos', 'sum'),
        TotalFalhas=('Falhas', 'sum'),
        # Count the number of runs remaining after outlier removal for this group
        NumRunsRemaining=('Rodada', 'count')
    ).reset_index()

    # Fill NaN std dev with 0 if there's only one data point remaining in a subset after outlier removal
    grouped_df['StdDevLatencyAcrossRuns'] = grouped_df['StdDevLatencyAcrossRuns'].fillna(0)


    print("\n--- Aggregated Statistics per Test Scenario (after outlier removal): ---")
    print(grouped_df.head())
    print("---------------------------------------------------------------")


    # --- Plot 1: Average Latency vs. Clients (across runs, with error bars) ---
    print("\nGerando gráficos de Latência Média vs. Clientes (com barras de erro)...")
    for messages_val in unique_messages:
        plt.figure(figsize=(14, 8))
        plt.title(f'Latência Média vs. Clientes ({messages_val} Mensagens/Cliente, Rodadas Agregadas)')
        plt.xlabel('Número de Clientes Simultâneos')
        plt.ylabel('Latência Média (ms)')
        plt.grid(True)
        plt.xticks(unique_clients)

        for server_val in unique_servers:
            subset = grouped_df[(grouped_df['Servidores'] == server_val) &
                                (grouped_df['MensagensPorCliente'] == messages_val)]
            if not subset.empty:
                subset = subset.sort_values(by='Clientes')
                plt.errorbar(subset['Clientes'], subset['AvgLatencyAcrossRuns'],
                             yerr=subset['StdDevLatencyAcrossRuns'],
                             marker='o', linestyle='-', capsize=5, label=f'{server_val} Servidores')

        plt.legend(title='Servidores')
        plt.tight_layout()
        plot_path = os.path.join(os.path.dirname(csv_file), f'avg_latency_vs_clients_msg{messages_val}_cleaned.png')
        plt.savefig(plot_path)
        plt.show()
        print(f"Gráfico salvo em: {plot_path}")


    # --- Plot 2: Average Latency vs. Servidores (across runs, with error bars) ---
    print("\nGerando gráficos de Latência Média vs. Servidores (com barras de erro)...")
    for messages_val in unique_messages:
        plt.figure(figsize=(14, 8))
        plt.title(f'Latência Média vs. Servidores ({messages_val} Mensagens/Cliente, Rodadas Agregadas)')
        plt.xlabel('Número de Servidores (Réplicas)')
        plt.ylabel('Latência Média (ms)')
        plt.grid(True)
        plt.xticks(unique_servers)

        for client_val in unique_clients:
            subset = grouped_df[(grouped_df['Clientes'] == client_val) &
                                (grouped_df['MensagensPorCliente'] == messages_val)]
            if not subset.empty:
                subset = subset.sort_values(by='Servidores')
                plt.errorbar(subset['Servidores'], subset['AvgLatencyAcrossRuns'],
                             yerr=subset['StdDevLatencyAcrossRuns'],
                             marker='o', linestyle='-', capsize=5, label=f'{client_val} Clientes')

        plt.legend(title='Clientes')
        plt.tight_layout()
        plot_path = os.path.join(os.path.dirname(csv_file), f'avg_latency_vs_servers_msg{messages_val}_cleaned.png')
        plt.savefig(plot_path)
        plt.show()
        print(f"Gráfico salvo em: {plot_path}")

    # --- Plot 3: Successes and Failures Summary ---
    print("\nGerando gráficos de Sucessos e Falhas Totais...")
    for messages_val in unique_messages:
        msg_subset = grouped_df[grouped_df['MensagensPorCliente'] == messages_val].copy() # Usar .copy() para evitar SettingWithCopyWarning
        msg_subset['Scenario'] = 'S' + msg_subset['Servidores'].astype(str) + '-C' + msg_subset['Clientes'].astype(str)
        msg_subset = msg_subset.sort_values(by=['Servidores', 'Clientes'])

        if not msg_subset.empty:
            plt.figure(figsize=(16, 8))
            width = 0.35

            x = range(len(msg_subset))

            plt.bar([i - width/2 for i in x], msg_subset['TotalSucessos'], width, label='Total Sucessos', color='lightgreen')
            plt.bar([i + width/2 for i in x], msg_subset['TotalFalhas'], width, label='Total Falhas', color='salmon')

            plt.title(f'Total de Sucessos e Falhas em Rodadas Limpas ({messages_val} Mensagens/Cliente)')
            plt.xlabel('Cenário (Servidores-Clientes)')
            plt.ylabel('Contagem Total')
            plt.xticks(x, msg_subset['Scenario'], rotation=45, ha='right')
            plt.legend()
            plt.grid(axis='y')
            plt.tight_layout()
            plot_path = os.path.join(os.path.dirname(csv_file), f'total_success_failure_msg{messages_val}_cleaned.png')
            plt.savefig(plot_path)
            plt.show()
            print(f"Gráfico salvo em: {plot_path}")

    print("\n--- Geração de gráficos concluída! ---")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Gera gráficos a partir do CSV de resultados de teste de carga.")
    parser.add_argument("--csv", type=str, default="full_load_test_results.csv",
                        help="Caminho para o arquivo CSV consolidado gerado pelo cliente Go.")
    parser.add_argument("--z-threshold", type=float, default=3.0,
                        help="Limiar do Z-score para detecção de outliers. Pontos com |Z-score| > threshold são removidos. (Padrão: 3.0)")
    args = parser.parse_args()

    try:
        import pandas
        import matplotlib.pyplot
        from scipy import stats # Verifica se scipy está disponível
    except ImportError:
        print("Erro: As bibliotecas 'pandas', 'matplotlib' e 'scipy' não estão instaladas.")
        print("Por favor, instale-as usando: pip install pandas matplotlib scipy")
        exit(1)

    plot_results(args.csv, args.z_threshold)
