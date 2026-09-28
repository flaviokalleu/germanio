#!/usr/bin/env bash
# Suíte oficial de benchmarks do Germanio (bench/README.md).
#
#   scripts/bench.sh            micro + aplicação (go test -bench), 6 repetições
#   scripts/bench.sh stress     também o stress: 1k e 10k conexões, Germanio e Go direto
#
# O resultado vai para bench/resultados/<data>-<commit>.txt com o contexto
# obrigatório: hardware, SO, Go, versão do Germanio, commit, comando, dataset.
set -euo pipefail
cd "$(dirname "$0")/.."

modo="${1:-}"
count="${COUNT:-6}"
commit="$(git rev-parse --short HEAD)"
sujo="$(git status --porcelain | grep -q . && echo " (com alterações locais)" || true)"
mkdir -p bench/resultados
out="bench/resultados/$(date +%Y-%m-%d)-${commit}.txt"
tmp="$(mktemp -d)"
gepid=""; gopid=""
pge="${PORTA_GE:-18380}"; pgo="${PORTA_GO:-18381}"
trap 'kill $gepid $gopid 2>/dev/null || true; rm -rf "$tmp"' EXIT

{
  echo "# Germanio — benchmarks"
  echo "data:      $(date -Iseconds)"
  echo "commit:    ${commit}${sujo}"
  echo "germanio:  $(go run ./cmd/ge --version 2>/dev/null | awk '{print $2}')"
  echo "go:        $(go version)"
  echo "so:        $(uname -srmo)"
  echo "cpu:       $(grep -m1 'model name' /proc/cpuinfo | cut -d: -f2 | xargs) ($(nproc) núcleos)"
  echo "memória:   $(grep MemTotal /proc/meminfo | awk '{print $2/1024/1024 " GB"}')"
  echo "governor:  $(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo n/d)"
  echo "dataset:   bench/testdata (clientes: 1000 registros semeados; sintético: 10/100/1000 dados)"
  echo "comando:   go test -run '^\$' -bench . -benchmem -count ${count} ./bench/"
  echo
} > "$out"

go test -run '^$' -bench . -benchmem -count "$count" ./bench/ 2>&1 | grep -v '^\[germanio\]' >> "$out"

if [ "$modo" = "stress" ]; then
  go build -o "$tmp/germanio" .
  go build -o "$tmp/baseline" ./bench/baseline/servidor
  go build -o "$tmp/stress" ./bench/stress
  cp bench/testdata/clientes.ge "$tmp/inicio.ge"
  for p in "$pge" "$pgo"; do
    if ss -ltn | grep -q ":$p "; then echo "porta $p ocupada (use PORTA_GE/PORTA_GO)" >&2; exit 1; fi
  done
  (cd "$tmp" && GERMANIO_SQLITE="$tmp/ge.db" exec ./germanio run inicio.ge "$pge" > "$tmp/ge.log" 2>&1) &
  gepid=$!
  "$tmp/baseline" -addr 127.0.0.1:"$pgo" -db "$tmp/go.db" > "$tmp/go.log" 2>&1 &
  gopid=$!
  sleep 3
  for i in $(seq 1 1000); do
    body="{\"nome\":\"Cliente $i\",\"email\":\"c$i@exemplo.com\",\"cidade\":\"Recife\"}"
    curl -s -o /dev/null -XPOST -H 'Content-Type: application/json' -d "$body" localhost:$pge/_ge/api/clientes
    curl -s -o /dev/null -XPOST -H 'Content-Type: application/json' -d "$body" localhost:$pgo/clientes
  done
  {
    echo
    echo "## stress (gerador em ciclo fechado; mesmo computador para servidor e gerador)"
    echo "comando: bench/stress -c <conexões> -d ${DURACAO:-20s} -url <listar página 3>"
  } >> "$out"
  for c in ${CONEXOES:-1000 10000}; do
    "$tmp/stress" -rotulo germanio -c "$c" -d "${DURACAO:-20s}" -pid "$gepid" -url "http://127.0.0.1:$pge/_ge/api/clientes?page=3" >> "$out"
    "$tmp/stress" -rotulo go -c "$c" -d "${DURACAO:-20s}" -pid "$gopid" -url "http://127.0.0.1:$pgo/clientes?page=3" >> "$out"
  done
fi

echo "resultado em $out"
