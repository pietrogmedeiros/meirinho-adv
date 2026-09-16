// hearing-service: recebe o áudio da audiência, guarda no object storage e
// dispara o pipeline (transcrição -> análise) publicando audio.uploaded.
//
// O serviço não transcreve nem analisa: ele só é dono do dado da audiência e
// do ponto de entrada. Quem faz o trabalho pesado são os workers.
package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/pietromedeiros/meirinho/internal/domain"
	"github.com/pietromedeiros/meirinho/internal/platform/authn"
	"github.com/pietromedeiros/meirinho/internal/platform/config"
	"github.com/pietromedeiros/meirinho/internal/platform/db"
	"github.com/pietromedeiros/meirinho/internal/platform/events"
	"github.com/pietromedeiros/meirinho/internal/platform/httpx"
	"github.com/pietromedeiros/meirinho/internal/platform/logging"
	"github.com/pietromedeiros/meirinho/internal/platform/service"
	"github.com/pietromedeiros/meirinho/internal/platform/storage"
)

// 300 MB cobre uma audiência longa gravada em celular.
const maxAudioBytes = 300 << 20

type api struct {
	store   *store
	bus     *events.Bus
	storage *storage.Client
}

func main() {
	rt, cleanup := service.Bootstrap("hearing-service")
	defer cleanup()

	pool, err := db.Open(rt.Ctx, config.MustString("DATABASE_URL"))
	if err != nil {
		rt.Fatal("abrir banco", err)
	}
	defer pool.Close()
	if err := pool.Aguardar(rt.Ctx, 60); err != nil {
		rt.Fatal("aguardar postgres", err)
	}

	bus := events.New(config.MustString("REDIS_ADDR"), rt.Log)
	defer bus.Close()
	if err := bus.Aguardar(rt.Ctx, 60); err != nil {
		rt.Fatal("aguardar redis", err)
	}

	st, err := storage.New(rt.Ctx, storage.Config{
		Endpoint:  config.MustString("MINIO_ENDPOINT"),
		AccessKey: config.MustString("MINIO_ACCESS_KEY"),
		SecretKey: config.MustString("MINIO_SECRET_KEY"),
		Bucket:    config.String("MINIO_BUCKET", "meirinho-audio"),
		UseSSL:    config.Bool("MINIO_USE_SSL", false),
		SSE:       config.Bool("MINIO_SSE_ENABLED", false),
	})
	if err != nil {
		rt.Fatal("cliente de storage", err)
	}
	if err := st.GarantirBucket(rt.Ctx, 60); err != nil {
		rt.Fatal("garantir bucket", err)
	}

	a := &api{store: &store{pool: pool}, bus: bus, storage: st}
	verifier := authn.NewVerifier(config.MustString("JWT_SECRET"))

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(httpx.Observability(rt.Log))
	r.Get("/health", httpx.Health)

	r.Route("/api/hearings", func(r chi.Router) {
		r.Use(authn.Middleware(verifier))
		r.Get("/", a.listar)
		r.Post("/", a.upload)
		r.Get("/{id}", a.detalhe)
	})

	if err := rt.ServirHTTP(config.String("PORT", "8082"), r); err != nil {
		rt.Fatal("servidor http", err)
	}
}

func (a *api) listar(w http.ResponseWriter, r *http.Request) {
	itens, err := a.store.listar(r.Context(), authn.TenantDo(r.Context()))
	if err != nil {
		logging.From(r.Context()).Error("listar audiências", "err", err)
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível listar")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"itens": itens})
}

// upload recebe multipart: campo `audio` (arquivo), `area_do_direito`, `titulo`.
func (a *api) upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := authn.TenantDo(ctx)
	log := logging.From(ctx)

	r.Body = http.MaxBytesReader(w, r.Body, maxAudioBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "upload_invalido", "arquivo ausente ou maior que o limite de 300 MB")
		return
	}

	area, ok := domain.ParseArea(r.FormValue("area_do_direito"))
	if !ok {
		httpx.Fail(w, http.StatusBadRequest, "area_invalida",
			"área do direito inválida (civel, familia, criminal, trabalhista, administrativo)")
		return
	}

	arquivo, header, err := r.FormFile("audio")
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "audio_obrigatorio", "envie o arquivo de áudio no campo 'audio'")
		return
	}
	defer arquivo.Close()

	titulo := strings.TrimSpace(r.FormValue("titulo"))
	if titulo == "" {
		titulo = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}

	// Prefixo por tenant: mesmo dentro do bucket o dado fica segregado, o que
	// simplifica expurgo por retenção e auditoria depois.
	objectKey := fmt.Sprintf("%s/%s%s", tenantID, uuid.NewString(), filepath.Ext(header.Filename))

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := a.storage.Upload(ctx, objectKey, arquivo, header.Size, contentType); err != nil {
		log.Error("upload para storage", "err", err)
		httpx.Fail(w, http.StatusBadGateway, "storage_indisponivel", "não foi possível guardar o áudio")
		return
	}

	aud, err := a.store.criar(ctx, tenantID, titulo, objectKey, header.Filename, area)
	if err != nil {
		log.Error("criar audiência", "err", err)
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível registrar a audiência")
		return
	}

	if err := a.bus.Publish(ctx, domain.StreamAudioUploaded, domain.AudioUploaded{
		HearingID: aud.ID,
		TenantID:  tenantID,
		ObjectKey: objectKey,
		Area:      area,
	}); err != nil {
		// A audiência já está persistida; o pipeline pode ser reprocessado.
		// Melhor devolver 202 com o registro do que perder o upload.
		log.Error("publicar audio.uploaded", "err", err, "hearing_id", aud.ID)
	}

	httpx.JSON(w, http.StatusAccepted, aud)
}

func (a *api) detalhe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	aud, objectKey, err := a.store.buscar(ctx, authn.TenantDo(ctx), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Fail(w, http.StatusNotFound, "nao_encontrada", "audiência não encontrada")
		return
	}

	// URL pré-assinada de vida curta: o bucket nunca é público.
	if url, err := a.storage.URLAssinada(ctx, objectKey, 15*time.Minute); err == nil {
		aud.AudioURL = url
	} else {
		logging.From(ctx).Warn("não foi possível assinar url do áudio", "err", err)
	}

	httpx.JSON(w, http.StatusOK, aud)
}
