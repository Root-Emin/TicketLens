#!/usr/bin/env bash
#
# start.sh — TicketLens monorepo development runner
#
# Backend (Go + Docker infra) ve frontend (Next.js) servislerini tek komutla
# ayağa kaldırır. Backend tarafı backend/dev.sh'a delege edilir.
#
# Kullanım:
#   ./start.sh            — Full stack: infra + migration + backend + frontend
#   ./start.sh backend    — Sadece backend (infra + migration + server)
#   ./start.sh frontend   — Sadece frontend (Next.js dev server)
#   ./start.sh infra      — Sadece altyapı (Postgres, Redis, Kafka) + migration
#   ./start.sh migrate    — Sadece migration
#   ./start.sh install    — Frontend bağımlılıklarını kur (npm install)
#   ./start.sh down       — Her şeyi durdur (backend + frontend + Docker)
#   ./start.sh logs       — Docker servis loglarını izle
#   ./start.sh clean      — Infra + volume + build artefaktlarını temizle
#   ./start.sh help       — Yardım
#
set -euo pipefail

# ─── Configuration ────────────────────────────────────────────────────────────
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"

BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-3000}"
# The backend base URL is server-side only: the browser never talks to :8080
# directly, it goes through the Next proxy. So this is API_URL, not
# NEXT_PUBLIC_API_URL — a NEXT_PUBLIC_ name would inline the internal address
# into the client bundle for no reason.
API_URL="${API_URL:-http://localhost:$BACKEND_PORT}"

BACKEND_PID=""
FRONTEND_PID=""
CLEANED_UP=0

# ─── Colors ───────────────────────────────────────────────────────────────────
RED=$'\033[0;31m'
GREEN=$'\033[0;32m'
YELLOW=$'\033[1;33m'
CYAN=$'\033[0;36m'
BLUE=$'\033[0;34m'
MAGENTA=$'\033[0;35m'
BOLD=$'\033[1m'
NC=$'\033[0m'

log_info()  { echo -e "${CYAN}[dev]${NC}   $*"; }
log_ok()    { echo -e "${GREEN}[dev]${NC}   $*"; }
log_warn()  { echo -e "${YELLOW}[dev]${NC}   $*"; }
log_error() { echo -e "${RED}[dev]${NC}   $*"; }
log_step()  { echo -e "\n${BOLD}━━━ $* ━━━${NC}"; }

# ─── Helpers ──────────────────────────────────────────────────────────────────

require_backend() {
    if [[ ! -x "$BACKEND_DIR/dev.sh" ]]; then
        log_error "backend/dev.sh bulunamadı veya çalıştırılabilir değil."
        exit 1
    fi
}

require_frontend() {
    if [[ ! -f "$FRONTEND_DIR/package.json" ]]; then
        log_error "frontend/package.json bulunamadı."
        exit 1
    fi
    if ! command -v npm &>/dev/null; then
        log_error "npm bulunamadı. Node.js kurulu olmalı."
        exit 1
    fi
}

install_frontend_deps() {
    require_frontend
    if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
        log_info "Frontend bağımlılıkları kuruluyor (npm install)..."
        (cd "$FRONTEND_DIR" && npm install)
        log_ok "Frontend bağımlılıkları hazır"
    fi
}

# Verilen porttaki artık süreçleri temizle
free_port() {
    local port=$1
    lsof -ti:"$port" 2>/dev/null | xargs kill -9 2>/dev/null || true
}

# Bir sürecin çalışma dizini proje ağacının altında mı?
in_project_tree() {
    local pid=$1 cwd
    cwd=$(lsof -a -p "$pid" -d cwd -Fn 2>/dev/null | sed -n 's/^n//p' | head -1)
    [[ -n "$cwd" && ( "$cwd" == "$ROOT_DIR" || "$cwd" == "$ROOT_DIR"/* ) ]]
}

# Bu projeye ait host süreçlerini bul (air, air'in derlediği binary, next dev).
# Desen tek başına yeterince seçici değil — başka bir repoda da air ya da next
# koşuyor olabilir — o yüzden adayları çalışma dizinine göre eliyoruz.
project_pids() {
    local pid
    for pid in $(pgrep -f 'air -c \.air\.toml|tmp/server|next-server|next dev|npm run dev' 2>/dev/null); do
        if [[ "$pid" != "$$" ]] && in_project_tree "$pid"; then
            echo "$pid"
        fi
    done
    return 0
}

# Önce nazikçe TERM, 5 saniye içinde kapanmayana KILL
stop_pids() {
    local pid alive i
    (( $# > 0 )) || return 0

    for pid in "$@"; do kill -TERM "$pid" 2>/dev/null || true; done

    for i in $(seq 1 10); do
        alive=0
        for pid in "$@"; do
            if kill -0 "$pid" 2>/dev/null; then alive=1; fi
        done
        if [[ $alive -eq 0 ]]; then return 0; fi
        sleep 0.5
    done

    for pid in "$@"; do kill -KILL "$pid" 2>/dev/null || true; done
}

# Bir sürecin tüm alt ağacını sonlandır (air → server binary, npm → next gibi)
kill_tree() {
    local pid=$1 child
    [[ -n "$pid" ]] || return 0
    for child in $(pgrep -P "$pid" 2>/dev/null || true); do
        kill_tree "$child"
    done
    kill -TERM "$pid" 2>/dev/null || true
}

# Alt sürecin çıktısını renkli etiketle öne alarak akıt
prefix_output() {
    local color=$1 label=$2
    awk -v c="$color" -v l="$label" -v nc="$NC" '{ printf "%s%s%s %s\n", c, l, nc, $0; fflush() }'
}

install_traps() {
    trap 'cleanup; exit 0' INT TERM
    trap cleanup EXIT
}

cleanup() {
    [[ $CLEANED_UP -eq 1 ]] && return 0
    CLEANED_UP=1

    echo ""
    log_step "Kapatılıyor"

    if [[ -n "$FRONTEND_PID" ]]; then
        log_info "Frontend durduruluyor..."
        kill_tree "$FRONTEND_PID"
    fi
    if [[ -n "$BACKEND_PID" ]]; then
        log_info "Backend durduruluyor..."
        kill_tree "$BACKEND_PID"
    fi

    sleep 1
    free_port "$BACKEND_PORT"
    free_port "$FRONTEND_PORT"

    log_ok "Uygulama süreçleri durduruldu"
    log_info "Docker servisleri hâlâ çalışıyor — durdurmak için: ${BOLD}./start.sh down${NC}"
}

# ─── Servis başlatıcılar ──────────────────────────────────────────────────────

start_backend_bg() {
    require_backend
    free_port "$BACKEND_PORT"

    (
        cd "$BACKEND_DIR"
        ./dev.sh server 2>&1 | prefix_output "$BLUE" "[backend] "
    ) &
    BACKEND_PID=$!
    log_ok "Backend başlatıldı (pid $BACKEND_PID) → http://localhost:$BACKEND_PORT"
}

start_frontend_bg() {
    require_frontend
    install_frontend_deps
    free_port "$FRONTEND_PORT"

    (
        cd "$FRONTEND_DIR"
        export API_URL="$API_URL"
        npm run dev -- --port "$FRONTEND_PORT" 2>&1 | prefix_output "$MAGENTA" "[frontend]"
    ) &
    FRONTEND_PID=$!
    log_ok "Frontend başlatıldı (pid $FRONTEND_PID) → http://localhost:$FRONTEND_PORT"
}

# İki süreçten biri ölene kadar bekle; ölürse hepsini kapat
supervise() {
    echo ""
    log_step "Çalışıyor — durdurmak için Ctrl+C"
    echo -e "  ${BOLD}Frontend${NC}   http://localhost:$FRONTEND_PORT"
    echo -e "  ${BOLD}API${NC}        http://localhost:$BACKEND_PORT"
    echo -e "  ${BOLD}Health${NC}     http://localhost:$BACKEND_PORT/health/ready"
    echo -e "  ${BOLD}Metrics${NC}    http://localhost:$BACKEND_PORT/metrics"
    echo -e "  ${BOLD}Kafka UI${NC}   http://localhost:8090"
    echo ""

    while true; do
        if [[ -n "$BACKEND_PID" ]] && ! kill -0 "$BACKEND_PID" 2>/dev/null; then
            log_error "Backend süreci sonlandı."
            BACKEND_PID=""
            break
        fi
        if [[ -n "$FRONTEND_PID" ]] && ! kill -0 "$FRONTEND_PID" 2>/dev/null; then
            log_error "Frontend süreci sonlandı."
            FRONTEND_PID=""
            break
        fi
        sleep 1
    done
}

# ─── Komutlar ─────────────────────────────────────────────────────────────────

cmd_all() {
    log_step "TicketLens full stack"

    # Altyapı + migration (backend/dev.sh infra ikisini de yapar)
    require_backend
    (cd "$BACKEND_DIR" && ./dev.sh infra)

    install_traps
    start_backend_bg
    start_frontend_bg
    supervise
}

cmd_backend() {
    require_backend
    install_traps
    (cd "$BACKEND_DIR" && ./dev.sh infra)
    start_backend_bg
    supervise
}

cmd_frontend() {
    install_traps
    start_frontend_bg
    supervise
}

cmd_infra()      { require_backend; (cd "$BACKEND_DIR" && ./dev.sh infra); }
cmd_classifier() { require_backend; (cd "$BACKEND_DIR" && ./dev.sh classifier); }
cmd_migrate() { require_backend; (cd "$BACKEND_DIR" && ./dev.sh migrate); }
# Host süreçleri (air + Go binary, next dev) — Docker'dan bağımsız çalışıyorlar,
# bu yüzden `docker compose down` tek başına localhost'u susturmuyor.
stop_local_stack() {
    log_step "Uygulama süreçleri durduruluyor"

    local pids=()
    # shellcheck disable=SC2207
    pids=($(project_pids))

    if (( ${#pids[@]} > 0 )); then
        log_info "Proje süreçleri bulundu: ${pids[*]}"
        # air'i önce indir: yaşarsa binary'i anında yeniden doğurur.
        stop_pids "${pids[@]}"
    else
        log_info "Çalışan proje süreci yok"
    fi

    # Emniyet kemeri: ağaçtan kopmuş ya da önceki koşudan artakalan port sahipleri.
    local port
    for port in "$BACKEND_PORT" "$FRONTEND_PORT"; do
        if lsof -ti:"$port" &>/dev/null; then
            log_warn "Port $port hâlâ dolu — zorla boşaltılıyor"
            free_port "$port"
        fi
    done

    log_ok "Uygulama süreçleri durduruldu (:$BACKEND_PORT, :$FRONTEND_PORT boşta)"
}

cmd_down() {
    require_backend
    stop_local_stack
    (cd "$BACKEND_DIR" && ./dev.sh down)
    log_ok "Her şey kapandı"
}

cmd_logs()    { require_backend; (cd "$BACKEND_DIR" && ./dev.sh logs); }

cmd_install() {
    require_frontend
    log_step "Frontend bağımlılıkları"
    (cd "$FRONTEND_DIR" && npm install)
    log_ok "Tamamlandı"
}

cmd_clean() {
    require_backend
    stop_local_stack
    log_step "Temizlik"
    (cd "$BACKEND_DIR" && ./dev.sh clean)
    if [[ -d "$FRONTEND_DIR/.next" ]]; then
        rm -rf "$FRONTEND_DIR/.next"
        log_ok "Temizlendi: frontend/.next"
    fi
}

cmd_help() {
    echo -e "${BOLD}TicketLens monorepo development runner${NC}"
    echo ""
    echo "Kullanım: ./start.sh [komut]"
    echo ""
    echo "Komutlar:"
    echo -e "  ${GREEN}(varsayılan)${NC}  Full stack: infra + migration + backend + frontend"
    echo -e "  ${GREEN}backend${NC}       Sadece backend (infra + migration + hot-reload server)"
    echo -e "  ${GREEN}frontend${NC}      Sadece frontend (Next.js dev server)"
    echo -e "  ${GREEN}infra${NC}         Sadece altyapı (Postgres, Redis, Kafka) + migration"
    echo -e "  ${GREEN}classifier${NC}    backend/ml model servisini derle ve başlat (opsiyonel)"
    echo -e "  ${GREEN}migrate${NC}       Sadece migration"
    echo -e "  ${GREEN}install${NC}       Frontend bağımlılıklarını kur"
    echo -e "  ${GREEN}down${NC}          Her şeyi durdur (backend + frontend + Docker)"
    echo -e "  ${GREEN}logs${NC}          Docker servis loglarını izle"
    echo -e "  ${GREEN}clean${NC}         Infra + volume + build artefaktlarını temizle"
    echo -e "  ${GREEN}help${NC}          Bu yardım metni"
    echo ""
    echo "Ortam değişkenleri:"
    echo "  BACKEND_PORT=8080"
    echo "  FRONTEND_PORT=3000"
    echo "  API_URL=http://localhost:8080   (sunucu tarafı; client'a sızmaz)"
    echo "  CLASSIFIER_URL                  (boş = keyword stub)"
}

# ─── Main ─────────────────────────────────────────────────────────────────────

case "${1:-}" in
    backend)  cmd_backend  ;;
    frontend) cmd_frontend ;;
    infra)    cmd_infra    ;;
    classifier) cmd_classifier ;;
    migrate)  cmd_migrate  ;;
    install)  cmd_install  ;;
    down)     cmd_down     ;;
    logs)     cmd_logs     ;;
    clean)    cmd_clean    ;;
    help|-h|--help) cmd_help ;;
    *)        cmd_all      ;;
esac
