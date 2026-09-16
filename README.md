# Meirinho

Assistente para advogado autônomo brasileiro. Duas frentes:

- **Audiência** — o advogado sobe o áudio; o sistema transcreve, analisa à luz da
  área do direito e devolve resumo, sugestão estratégica e pontos críticos.
- **Processo** — o advogado cadastra o CNJ; o sistema acompanha as movimentações,
  classifica a urgência de cada uma e avisa quando algo abre prazo.

## Subir

```bash
cp .env.example .env   # opcional: os padrões já funcionam
docker compose up --build -d
```

A API fica em `http://localhost:8080`. Sem nenhuma credencial externa o pipeline
roda de ponta a ponta em modo mock — transcrição, análise e provider de tribunal
são simulados. Para análise real, `ANALYZER=claude` + `ANTHROPIC_API_KEY`.

> Em máquina com pouca memória, `docker compose build` em paralelo pode estourar
> a RAM da VM do Docker (são 8 binários Go). Nesse caso, construa em série:
> `for s in $(docker compose config --services); do docker compose build $s; done`

## Pipeline

```
                         ┌─ audio.uploaded ──> transcription-worker ─┐
hearing-service ─────────┘                                          │
                                                       audio.transcribed
                                                                    │
                                          analysis-service <────────┘
                                                    │
                                          hearing.analyzed ──┐
                                                             ├──> notification-service
                                         movement.classified ─┘
                                                    ▲
                                       classification-worker
                                                    ▲
                                          movement.received
                                                    │
                                     process-monitor-service (poller)
```

O barramento é Redis Streams com consumer group por serviço: réplicas do mesmo
serviço dividem a carga, serviços diferentes recebem cada um sua cópia.

## Serviços

| serviço | porta | papel |
|---|---|---|
| gateway (nginx) | 8080 | porta única da API |
| auth-service | 8081 | cadastro, login, JWT — um advogado é um tenant |
| hearing-service | 8082 | dono da audiência e do upload |
| analysis-service | 8083 | transcrição → resumo e sugestão |
| notification-service | 8084 | caixa de entrada; consome os dois streams finais |
| process-monitor-service | 8085 | carteira de processos + poller |
| transcription-worker | 8086 | áudio → texto |
| classification-worker | 8087 | movimentação → urgência |

## Decisões que não são óbvias no código

**Isolamento por tenant é do banco, não da aplicação.** As tabelas têm ROW LEVEL
SECURITY e as políticas comparam `tenant_id` com `app.current_tenant`. Os serviços
conectam como `meirinho_app`, papel sem `BYPASSRLS`. Uma query que esqueça o
`WHERE tenant_id` não devolve linha de outro tenant — ela não devolve nada. O
único caminho para dado de tenant é `db.TenantTx`.

**Na dúvida, o classificador alerta.** O risco não é simétrico: um alerta
desnecessário custa trinta segundos de leitura; uma intimação classificada como
"nenhuma" custa um prazo. Quando a confiança fica abaixo de `CLASSIFY_MIN_CONFIANCA`
ou a urgência vem fora do domínio, a urgência é elevada ao piso (`CLASSIFY_PISO`)
e a movimentação é marcada com `fallback_aplicado`. A regra nunca rebaixa: se o
modelo classificou acima do piso com pouca confiança, a classificação dele fica.
O `fallback_aplicado` também fura o filtro de urgência mínima da notificação —
ali "baixa" significa "não sei", não "não importa".

**Idempotência é do banco.** O provider reenvia o mesmo andamento a cada consulta;
o índice único `(process_id, provider_event_id)` com `ON CONFLICT DO NOTHING` é o
que impede evento duplicado de entrar no barramento — não uma checagem otimista.

**O poller varre por tenant.** Não por escolha estética: a lista de processos está
sob RLS, então só é visível dentro de uma transação com tenant setado. O efeito
colateral é bom — um tenant com problema não interrompe a varredura dos outros.

**Mock é o padrão em toda fronteira externa.** `docker compose up` numa máquina
sem chave de API e sem credencial de tribunal precisa exercitar o pipeline inteiro,
senão nada disso é testável localmente.

## Ainda não existe

- `apps/` — nenhum front-end.
- Provider real de tribunal (DataJud/PJe): a interface `Provider` é o ponto de
  encaixe; só o mock está implementado.
- Envio real de e-mail e WhatsApp — os canais existem atrás de flag desligada.
- Job de expurgo por retenção: `retencao_dias` existe na tabela, sem rotina que o aplique.
- Testes automatizados.
