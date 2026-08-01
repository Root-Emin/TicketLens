# masterfabric-go — Backend Mimari Rehberi

> Bu doküman bir kod incelemesi (code review) değildir. Amacı, backend mimarisini
> **öğretmektir**: hangi dosya neden var, hangi katmana ait, sistem çalışırken ne
> yapıyor, silinirse ne kırılır.

---

## 1. Bu Proje Ne Yapıyor?

`masterfabric-go`, tek başına bir "ürün API'si" değil — **başka API'leri yöneten bir platformdur**.
En yalın tanımı: **çok kiracılı (multi-tenant), RBAC tabanlı, kendi kendini konfigüre eden bir API Gateway + SaaS kontrol düzlemi.**

İki farklı "iş" aynı binary içinde koşuyor:

| İş | Ne yapar | Örnek URL |
|---|---|---|
| **Control Plane** (Yönetim düzlemi) | Kullanıcı, organizasyon, workspace, app, API key, endpoint tanımı, politika yönetimi | `POST /api/v1/organizations/{id}/apps` |
| **Data Plane** (Trafik düzlemi) | Yukarıda *tanımlanmış* endpoint'lere gelen gerçek trafiği yakalar, politikayı uygular, backend'e yönlendirir | `GET /api/v1/products` |

**Gerçek hayat benzetmesi:** Bir **AVM yönetimi** düşün.
- *Control plane* = AVM yönetim ofisi. Kim hangi dükkânı açabilir, hangi dükkân hangi saatte çalışır, kimin anahtarı var — bunları kaydeder.
- *Data plane* = AVM'nin giriş kapısındaki güvenlik + yönlendirme. Gelen müşteriyi kimliğine bakıp doğru dükkâna yollar, yasaksa içeri almaz.

### Temel Kavram Hiyerarşisi

```
Organization (kiracı / şirket)
  └── Workspace (opsiyonel ara katman: "prod", "staging", "ekip-a")
        └── App (mantıksal uygulama: "mobil-app", "e-ticaret")
              ├── API Key   (makine-makine erişim anahtarı)
              └── Endpoint  (yönetilen API tanımı: GET /products)
                    └── Policy (izin, rate limit, şema, auth türü)
```

---

## 2. Mimari Model: Clean / Hexagonal Architecture

Proje 4 halkalı bir soğan yapısındadır. **En kritik kural: bağımlılıklar sadece içeri doğru akar.**

```
        ┌─────────────────────────────────────────────┐
        │  INFRASTRUCTURE (dış dünya)                 │   ← Postgres, Kafka, HTTP, Redis, JWT
        │   ┌───────────────────────────────────────┐ │
        │   │  APPLICATION (use case / senaryo)     │ │   ← "Kullanıcı kaydet", "App oluştur"
        │   │   ┌─────────────────────────────────┐ │ │
        │   │   │  DOMAIN (iş kuralları)          │ │ │   ← User, Org, Endpoint + arayüzler
        │   │   │  * Sıfır dış bağımlılık *       │ │ │
        │   │   └─────────────────────────────────┘ │ │
        │   └───────────────────────────────────────┘ │
        └─────────────────────────────────────────────┘
                          ▲
                    SHARED (her katmanın kullandığı yardımcılar)
```

### Bağımlılık Kuralı Pratikte Ne Demek?

`internal/domain/iam/repository/user_repository.go` şunu der:

```go
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    GetByEmail(ctx context.Context, email string) (*model.User, error)
}
```

Bu bir **söz** (contract). "Kullanıcıyı bir yere kaydedeceğim" der ama **nereye** kaydedeceğini bilmez.
`internal/infrastructure/postgres/iam/user_repository.go` bu sözü PostgreSQL ile yerine getirir.

**Neden bu kadar uğraşılıyor?**
- Yarın Postgres yerine MongoDB'ye geçmek istersen sadece `infrastructure/` değişir, iş mantığı hiç dokunulmaz.
- Test yazarken gerçek veritabanı gerekmez; sahte (mock) bir `UserRepository` verirsin.
- **Benzetme:** Bir priz standardı gibi. Duvardaki priz (interface) neyin takılacağını bilmez; ütü de televizyon da takılabilir.

### Dependency Injection Nerede Yapılıyor?

Tek yerde: **`cmd/server/main.go` içindeki `buildDependencies()` fonksiyonu.**
Bu projede DI framework'ü (wire, fx vb.) yok — her şey elle, açıkça bağlanıyor. Bu iyi bir tercih: bağlantı zincirini tek dosyada okuyabilirsin.

---

## 3. Bir İsteğin Yaşam Döngüsü (En Önemli Bölüm)

`POST /api/v1/organizations/{orgId}/apps` isteğini takip edelim:

```
1. TARAYICI/POSTMAN
        │
        ▼
2. cmd/server/main.go → http.Server
        │
        ▼
3. internal/infrastructure/http/router/router.go  ← Chi router
        │
        ├─ middleware.RequestID      → Her isteğe benzersiz ID ver (X-Request-ID)
        ├─ middleware.Logging        → İsteği logla (süre, status)
        ├─ middleware.Recoverer      → Panik olursa çökme, 500 dön
        ├─ middleware.MaxBodyBytes   → 1MB'dan büyük body'yi reddet
        ├─ cors.Handler              → Tarayıcı origin kontrolü
        ├─ middleware.JWTAuth        → Token'ı doğrula, userID'yi context'e koy
        ├─ middleware.TenantResolver → Hangi organizasyon? (header/JWT/subdomain)
        ├─ gateway.Pipeline.Enforce  → Bu path yönetilen bir endpoint mi? (burada değil, geçer)
        └─ middleware.RequirePermission("app:write") → RBAC kontrolü
        │
        ▼
4. internal/infrastructure/http/handler/tenant/handler.go → CreateApp()
        │  (JSON'u parse et, valide et — İŞ MANTIĞI YOK)
        ▼
5. internal/application/tenant/usecase/create_app.go → Execute()
        │  (Org var mı? Aktif mi? Slug çakışıyor mu? → İŞ SENARYOSU BURADA)
        ▼
6. internal/domain/tenant/repository/app_repository.go (interface)
        │
        ▼
7. internal/infrastructure/postgres/tenant/app_repository.go → INSERT INTO apps
        │
        ▼
8. eventBus.Publish(TopicTenant, AppCreated{...})
        │
        ├─→ Kafka / in-process bus
        └─→ websocket.EventBridge → bağlı WebSocket istemcilerine anlık push
        │
        ▼
9. response.Created(w, appInfo) → 201 + JSON
```

**Her katmanın tek cümlelik görevi:**

| Katman | Görev | Yapmaması gereken |
|---|---|---|
| Middleware | Kimlik, güvenlik, gözlemlenebilirlik | İş kuralı çalıştırmak |
| Handler | HTTP ↔ Go dönüşümü | Veritabanına dokunmak, kural yazmak |
| Use Case | İş senaryosunu yürütmek | HTTP/SQL bilmek |
| Repository | Veriyi saklamak/getirmek | Karar vermek |
| Model | Veriyi ve değişmez kuralları taşımak | I/O yapmak |

---

## 4. Klasör Klasör Analiz

---

### 📁 `cmd/`

**Neden var?** Go standardı: çalıştırılabilir programların giriş noktaları burada durur.
Birden fazla binary üretmek istersen (`cmd/server`, `cmd/worker`, `cmd/cli`) her biri ayrı alt klasör olur.

**Sorumluluğu:** Programı **başlatmak** ve **parçaları birbirine bağlamak**. İş mantığı içermez.

**İletişimi:** Neredeyse *her* klasörle konuşur — çünkü hepsini birleştiren yer burasıdır. Ama tersi doğru değil: hiçbir paket `cmd/`'yi import etmez.

#### `cmd/server/main.go`
- **Amaç:** Uygulamanın giriş noktası. Composition Root (birleştirme kökü).
- **Neden var?** Config yükleme → logger → DB → Redis → Kafka → repository → use case → handler → router → HTTP server zincirini kuran tek yer.
- **Çalışma anındaki görevi:**
  1. `config.Load()` ile ortam değişkenlerini oku
  2. Postgres/Redis/Kafka bağlantılarını aç (**başarısız olursa çökmez, uyarı verip devam eder** — geliştirici dostu tasarım)
  3. `buildDependencies()` ile tüm nesneleri üret ve birbirine bağla
  4. HTTP sunucusunu ayağa kaldır
  5. SIGTERM/Ctrl+C gelince **graceful shutdown**: 30 sn içinde açık istekleri bitir, sonra kapan
- **İlişkili dosyalar:** `router.go` (Dependencies struct'ını doldurur), tüm `usecase/`, tüm `postgres/` repo'ları, `shared/config`
- **Silinirse:** Program derlenmez. Hiçbir şey çalışmaz. Bu dosya **kalbin kendisi.**
- **Katman:** Bootstrap / Composition Root
- **Benzetme:** Bir orkestranın **şefi**. Kendisi enstrüman çalmaz ama kimin ne zaman gireceğini o belirler.

> 💡 **Öğrenme notu:** `buildDependencies()` fonksiyonunu satır satır oku. Projeyi anlamanın en hızlı yolu budur — hangi nesne kimden besleniyor, tek bakışta görürsün.

---

### 📁 `internal/`

Go'da `internal` **özel bir kelimedir**: bu klasörün altındaki paketler yalnızca aynı modül içinden import edilebilir. Dışarıdan `github.com/masterfabric-go/masterfabric/internal/...` import etmek derleyici tarafından engellenir.

**Neden?** Dış dünyaya yanlışlıkla API sızdırmamak için. Tüm iç mimari burada saklı.

---

### 📁 `internal/domain/` — İş Kurallarının Kalbi

**Neden var?** İş dünyasının dilini koda çevirmek için. Burada "Postgres", "HTTP", "Kafka" kelimeleri **geçmez**.

**İçindeki dosya tipleri ve neden:**

| Alt klasör | Ne içerir | Neden ayrı |
|---|---|---|
| `model/` | Veri yapıları + değişmez kurallar (`IsActive()`) | Sistemin "isimleri" |
| `repository/` | Veri erişim **arayüzleri** (implementasyon yok!) | Bağımlılığı tersine çevirmek |
| `service/` | Domain servis **arayüzleri** (AuthService, RBACService) | Tek modele sığmayan kurallar |
| `event/` | Domain olayları (UserRegistered, AppCreated) | "Ne oldu"yu ilan etmek |

**İletişimi:** Domain **hiç kimseyi çağırmaz.** Herkes onu çağırır. Bu bilinçlidir.

**İstek buraya ne zaman gelir?** Doğrudan asla. Use case'ler üzerinden dolaylı gelir.

**Bounded Context'ler (DDD terimi — birbirinden bağımsız iş alanları):**

#### `internal/domain/iam/` — Identity & Access Management
- `model/user.go` — Kullanıcı varlığı. `PasswordHash` alanında `json:"-"` var → **JSON'a asla serialize edilmez.** Güvenlik açısından kritik detay.
- `model/role.go` — Rol, Permission, UserRole. Roller "organization" veya "app" kapsamında olabilir (`ScopeType`).
- `model/organization_user.go` — Kullanıcı-organizasyon üyeliği (active/invited/removed).
- `repository/*.go` — 3 arayüz: User, Role, OrgUser. `RoleRepository` en zengin olanı: izin ekleme/çıkarma, kullanıcıya rol atama, `GetUserPermissions`.
- `service/auth_service.go` — Şifre hash'leme + JWT üretme/doğrulama **sözleşmesi**.
- `service/rbac_service.go` — "Bu kullanıcının şu izni var mı?" sözleşmesi.
- `event/events.go` — UserRegistered, RoleAssigned vb.
- **Silinirse:** Kimlik doğrulama ve yetkilendirme tümüyle çöker. Tüm proje derlenmez.
- **Benzetme:** Bir binanın **kartlı geçiş sistemi tanımı**. Kimin hangi kata çıkabileceğinin kuralları.

#### `internal/domain/tenant/` — Kiracı Yönetimi
- `model/organization.go`, `model/workspace.go`, `model/app.go`, `model/api_key.go`
- `AppAPIKey.KeyHash` alanı da `json:"-"` — anahtarın kendisi **asla** DB'de veya response'ta düz metin durmaz.
- **Benzetme:** AVM'nin **dükkân kayıt defteri**. Hangi şirket, hangi katta, hangi dükkânda.

#### `internal/domain/apimanagement/` — Endpoint Yönetimi
- `model/endpoint.go` — **Projenin en özgün fikri.** Bir API endpoint'i burada *veri* olarak tutulur:
  ```go
  type Endpoint struct {
      Method         string   // "GET"
      Path           string   // "/products"
      BackendService string   // "product-service" veya "https://api.x.com"
      BackendAction  string   // "list" / "create" / "get"
      Schema         []byte   // JSON Schema (istek doğrulama)
      PIIMasking     bool     // Hassas alanları maskele
      Status         EndpointStatus
  }
  ```
  Yani **kod yazmadan, veritabanına satır ekleyerek yeni API endpoint'i tanımlayabiliyorsun.**
- `model/endpoint_policy.go` — O endpoint'e uygulanacak politika: gereken izin, rate limit, auth türü.
- **Benzetme:** Bir restoranın **menü kartı**. Yemekler kodda değil, menüde tanımlı; menüyü değiştirince mutfak yeniden inşa edilmiyor.

#### `internal/domain/audit/`
- `model/audit_log.go` — **Değiştirilemez (immutable)** denetim kaydı. Kim, ne zaman, neyi, hangi IP'den yaptı.
- `repository/audit_repository.go` — Sadece `Create` ve `List*` var. **`Update` ve `Delete` yok** — bu kasıtlı bir tasarım kararı. Denetim kaydı silinmez.
- **Benzetme:** Bir bankanın **kamera kaydı**. Sonradan oynanamaz.

#### `internal/domain/realtime/`
- `model/room.go` — WebSocket "oda" anahtarı: `org:{uuid}:app:{uuid}:channel:{isim}`. Kanal ismi regex ile doğrulanır (`^[a-zA-Z0-9_-]{1,64}$`) → **injection koruması.**
- `model/message.go` — İstemci→sunucu (`subscribe`, `ping`) ve sunucu→istemci (`pong`, `subscribed`, `error`) mesaj formatları.
- `service/hub.go` — Bağlantı yöneticisi arayüzü.
- **Benzetme:** Bir **telsiz kanalı sistemi**. Herkes aynı frekansı dinlemez; sadece kendi kanalını.

#### `internal/domain/gateway/interceptor.go`
- Gateway'in **eklenti (plugin) sözleşmesi**. `InterceptRequest` / `InterceptResponse`.
- `Chain` yapısı: istekleri **sırayla**, yanıtları **ters sırayla** işler (soğan modeli — tıpkı middleware gibi).
- **Benzetme:** Havalimanı güvenlik hattı. Gidişte X-ray → pasaport → bilet; dönüşte tam tersi.

---

### 📁 `internal/application/` — Senaryo Katmanı

**Neden var?** "Kullanıcı kaydolur" gibi bir senaryonun **adımlarını** tutmak için. Bir handler'a da model'e de sığmayan orkestrasyon burada.

**İçinde neden bu dosyalar var?**
- `dto/` — Data Transfer Object. Dış dünyanın gördüğü şekiller. **Neden ayrı?** Domain modelini asla doğrudan API'ye açmazsın; `PasswordHash` sızmasın diye. Ayrıca `validate:"required,email"` etiketleri burada.
- `usecase/` — **Her dosya = tek bir iş senaryosu.** Bu bilinçli: `create_app.go` dosyasını açtığında ne yaptığını tam olarak bilirsin.

**İletişimi:** Yukarıdan `handler/` çağırır; aşağıya `domain/repository` arayüzlerini ve `shared/events`'i çağırır. **`infrastructure/`'ı asla import etmez** (Postgres'i tanımaz).

**İstek buraya ne zaman gelir?** Handler JSON'u parse edip valide ettikten sonra.

#### Örnek: `internal/application/iam/usecase/register.go`
Çalışma anındaki görevi, adım adım:
1. `userRepo.GetByEmail()` → e-posta zaten var mı?
2. `auth.HashPassword()` → bcrypt ile hash'le
3. `userRepo.Create()` → kaydet
4. `eventBus.Publish(TopicIAM, UserRegistered{...})` → olayı duyur
5. `dto.UserInfo` döndür (şifre olmadan)
- **Silinirse:** `/api/v1/auth/register` çalışmaz, derleme kırılır.
- **Benzetme:** Bir **kontrol listesi (checklist)**. Pilot kalkış öncesi maddeleri sırayla işaretler.

#### Dikkat çeken use case'ler
- `iam/usecase/login.go` — Kullanıcıyı bulur, aktif mi bakar, şifreyi doğrular, JWT üretir. **Not:** Üretilen token'da `OrganizationID` ve `Permissions` boş — bunlar sonradan `TenantResolver` middleware'i ve `RBACService` tarafından çözülüyor.
- `tenant/usecase/manage_api_keys.go` — `crypto/rand` ile 32 byte rastgele anahtar üretir, `mf_` ön eki ekler, **SHA-256 hash'ini** kaydeder. Ham anahtar **sadece oluşturma anında bir kez** döner. Kaybedersen yenisini üretmen gerekir — doğru güvenlik pratiği.
- `apimanagement/usecase/define_endpoint.go` — Yeni bir yönetilen endpoint tanımlar. Bu çalıştıktan sonra gateway o path'i tanımaya başlar.
- `realtime/usecase/validate_connect.go` — WebSocket bağlantısı kurulmadan **önce** çalışır: app var mı, org'a ait mi, aktif mi, kullanıcının `app:read` izni var mı? Ancak hepsi geçerse el sıkışma (handshake) yapılır.

> ⚠️ **Mimari gözlem:** `tenant/usecase/create_org.go`, `create_workspace.go` ve `list_workspaces.go` dosyaları `shared/middleware` paketini import ediyor (context key'lerine erişmek için). Bu, application katmanının HTTP dünyasına sızmasıdır — Clean Architecture açısından küçük bir ihlal. Daha temizi: orgID'yi parametre olarak handler'dan geçirmek.

---

### 📁 `internal/infrastructure/` — Dış Dünya Adaptörleri

**Neden var?** Domain'in tanımladığı **arayüzleri gerçekten çalışan koda** çevirmek için. "Port & Adapter" mimarisindeki **adapter**'lar burada.

**Sorumluluğu:** Teknoloji bilgisi. SQL cümleleri, HTTP header'ları, Kafka topic'leri, JWT imzaları — hepsi burada hapsedilir.

**İletişimi:** `domain/` ve `application/`'ı import eder; onlar bunu asla import etmez.

---

#### 📁 `internal/infrastructure/http/router/router.go`
- **Amaç:** Tüm URL haritası. Hangi path hangi handler'a, hangi middleware zinciriyle gider.
- **Neden var?** Rotalar dağınık olsaydı "bu endpoint korumalı mı?" sorusuna cevap veremezdin. Tek dosyada toplu görüyorsun.
- **Kritik detay — middleware sırası:**
  ```go
  RequestID → Logging → Recoverer → MaxBodyBytes → CORS
     → [/api/v1/auth = KORUMASIZ]
     → [diğerleri: JWTAuth → TenantResolver → WebSocket → GatewayPipeline → RBAC]
  ```
  Bu sıra **rastgele değil**. `RequestID` en başta olmalı ki log'lar onu kullanabilsin. `Recoverer` erken olmalı ki alt katmanların paniğini yakalasın.
- **`Dependencies` struct'ı:** Router'ın ihtiyaç duyduğu her şeyin listesi. `main.go` bunu doldurur. Alanlar `nil` olabilir — o zaman ilgili rotalar kaydedilmez (veritabanısız çalışabilme özelliği).
- **Silinirse:** Sunucu ayağa kalkar ama hiçbir URL'ye cevap vermez.
- **Katman:** Routing / API
- **Benzetme:** Bir binanın **kat planı ve yönlendirme tabelaları**.

#### 📁 `internal/infrastructure/http/handler/`

**Bu klasör neden var?** HTTP protokolünü iş mantığından ayırmak için. Handler'ın tek işi: **çevir, çağır, çevir.**

Her handler'ın iskeleti aynı:
```go
func (h *Handler) CreateApp(w http.ResponseWriter, r *http.Request) {
    // 1. URL parametresini çöz
    // 2. JSON body'yi decode + validate et
    // 3. Use case'i çağır
    // 4. Hatayı veya sonucu JSON'a çevir
}
```

| Dosya | Sorumluluk |
|---|---|
| `iam/handler.go` | Register, Login, GetMe, ListUsers, GetUser, AssignRole |
| `tenant/handler.go` | Org/App/Workspace/APIKey CRUD |
| `apimanagement/handler.go` | Endpoint tanımlama, aktive/emekli etme, politika güncelleme |
| `audit/handler.go` | Denetim kaydı listeleme (sadece okuma) |
| `realtime/handler.go` | WebSocket yükseltme (upgrade) + istemci mesaj yönlendirme |
| `health/handler.go` | `/health/live` ve `/health/ready` probları |

- **`health/handler.go` neden özel?** Kubernetes/Docker bunları kullanır:
  - `Liveness` → "Süreç hayatta mı?" Hayır ise container yeniden başlatılır.
  - `Readiness` → "Trafik alabilir mi?" DB/Redis'i pingler. Hayır ise load balancer trafik göndermez.
  - Hata detayı **dışarı sızdırılmaz** (sadece "unhealthy"), iç hata log'a yazılır. Bilgi sızıntısı önlemi.
- **Silinirse:** İlgili API grubu 404 döner; router derlenmez.
- **Benzetme:** Bir kurumun **danışma masası**. Talebi anlar, doğru birime iletir, cevabı sana anlayacağın dilde söyler. Kendisi karar vermez.

#### 📁 `internal/infrastructure/postgres/`

**Neden var?** Domain'deki `repository` arayüzlerinin gerçek SQL implementasyonu.

**Ortak desen (her repo'da aynı):**
```go
type UserRepo struct { db *pgxpool.Pool }        // Bağlantı havuzu enjekte edilir

func (r *UserRepo) GetByID(...) (*model.User, error) {
    err := r.db.QueryRow(ctx, `SELECT ... WHERE id = $1`, id).Scan(&u.ID, ...)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, domainErr.New(domainErr.ErrNotFound, "user not found", nil)
    }
    ...
}
```
**Öğretici nokta:** `pgx.ErrNoRows` (teknoloji hatası) → `ErrNotFound` (domain hatası) dönüşümü burada yapılır. Böylece use case katmanı Postgres'in varlığından haberdar olmaz. Bu **hata çevirisi (error translation)** Clean Architecture'ın en sık atlanan detayıdır.

- **ORM kullanılmıyor** — ham SQL + `pgx`. Neden? Performans ve tam kontrol. Bedeli: daha çok kod.
- **Parametreli sorgular (`$1`, `$2`)** kullanılıyor → SQL injection koruması.
- **Silinirse:** `main.go` derlenmez; hiçbir veri okunup yazılamaz.
- **Katman:** Repository / Persistence
- **Benzetme:** Bir kütüphanenin **arşiv görevlisi**. "Şu kitabı getir" dersin; rafın nerede olduğunu sadece o bilir.

##### 📁 `internal/infrastructure/postgres/migrations/`
- **Amaç:** Veritabanı şemasının **sürüm kontrollü** tarihi. `goose` formatında (`-- +goose Up` / `-- +goose Down`).
- 12 migration var, sıralı: organizations → users → organization_users → roles → role_permissions → user_roles → apps → app_api_keys → app_endpoints → app_endpoint_policies → audit_logs → workspaces
- **Neden numaralı?** Sıra önemli. `apps` tablosu `organizations`'a foreign key ile bağlı; önce org tablosu var olmalı.
- **Öğretici detay:** `00012_create_workspaces.sql`, `apps` tablosuna sonradan `workspace_id` kolonu ekliyor — `NULL` kabul ederek. Bu **geriye dönük uyumlu (backward compatible) migration** tekniğidir: mevcut kayıtlar bozulmaz.
- **Silinirse:** Tablolar oluşmaz, tüm sorgular "relation does not exist" hatası verir.
- **Benzetme:** Bir binanın **tadilat ruhsatları dosyası**. Hangi duvar ne zaman yıkıldı, sırasıyla kayıtlı.

#### 📁 `internal/infrastructure/auth/`

##### `jwt_service.go`
- `domain/iam/service.AuthService` arayüzünü **bcrypt + HMAC-SHA256 JWT** ile gerçekler.
- `HashPassword` → bcrypt (yavaş olması **kasıtlı** — brute force'u pahalılaştırır)
- `ValidateToken` → imza algoritmasını **açıkça kontrol eder** (`SigningMethodHMAC` mi?). Bu, ünlü **"alg: none" JWT saldırısına** karşı zorunlu bir savunmadır.
- **Silinirse:** Login/register çalışmaz, hiçbir korumalı endpoint'e girilemez.
- **Benzetme:** **Pasaport basım ve doğrulama ofisi**.

##### `rbac_service.go`
- İzin kontrolü + **Redis önbelleği** (15 dk TTL).
- Neden cache? Her istekte DB'ye "bu kullanıcının izinleri ne?" diye sormak pahalı olur.
- `matchesPermission()` joker karakter destekler:
  - `*` → her şey (admin)
  - `app:*` → `app:read`, `app:write` hepsi
  - `*:read` → tüm okuma izinleri
- Rol değişince `InvalidateCache()` çağrılır (bkz. `assign_role.go`) — yoksa 15 dk boyunca eski izinler geçerli olurdu.
- **Silinirse:** Tüm yetki kontrolleri kırılır.
- **Benzetme:** **Kapı görevlisinin not defteri.** Her seferinde merkeze telefon etmek yerine sık gelenleri deftere yazıyor, ama liste güncellenince defteri yeniliyor.

#### 📁 `internal/infrastructure/kafka/`

| Dosya | Görev |
|---|---|
| `admin.go` | Başlangıçta topic'lerin var olduğundan emin ol (`EnsureTopics`) |
| `producer.go` | Olayları Kafka'ya yaz |
| `consumer.go` | Topic'lerden oku, `Envelope`'a çöz, handler'lara dağıt |
| `bus.go` | `events.EventBus` arayüzünü Kafka ile gerçekleyen sarmalayıcı |

- `bus.go` içindeki `camelToDotCase()` fonksiyonu güzel bir detay: Go struct adı `UserRegistered` → olay tipi `user.registered`. Reflection ile otomatik.
- **`KAFKA_ENABLED=false` ise** bu klasör hiç devreye girmez; `shared/events/inprocess.go` kullanılır. **Bu, mimarinin en öğretici yanı:** aynı `EventBus` arayüzü, iki farklı taşıyıcı.
- **Silinirse:** Kafka modu çalışmaz ama in-process modda uygulama ayakta kalır (yine de derleme kırılır çünkü `main.go` import ediyor).
- **Benzetme:** **Postane.** Mektubu (olay) alır, doğru posta kutusuna (topic) koyar; aboneler kutularını kontrol eder.

#### 📁 `internal/infrastructure/websocket/`

| Dosya | Görev |
|---|---|
| `hub.go` | Tüm bağlantıların ve odaların merkezi kaydı. `sync.RWMutex` ile eşzamanlılık korumalı |
| `client.go` | Tek bir bağlantı: `readPump` (oku) + `writePump` (yaz) goroutine çifti |
| `session.go` | Bir bağlantının yaşam döngüsünü başlatır |
| `upgrader.go` | HTTP → WebSocket protokol yükseltmesi + Origin kontrolü |
| `event_bridge.go` | **Kritik köprü:** Event bus'taki domain olaylarını WebSocket odalarına yayınlar |

- **`event_bridge.go` neden çok önemli?** Bu dosya sayesinde birisi yeni bir App oluşturduğunda, o organizasyona bağlı tüm WebSocket istemcileri **anında** haberdar olur. Backend'den frontend'e canlı veri akışının kalbi.
- `client.go` içindeki iki-goroutine deseni (`readPump`/`writePump`) **standart Gorilla WebSocket pratiğidir**: bir bağlantıya aynı anda birden fazla goroutine yazamaz, o yüzden yazma işi tek bir goroutine'de merkezîleştirilir.
- **Ping/Pong:** 30 saniyede bir ping atılır, 60 saniye pong gelmezse bağlantı ölü sayılır. Ağdaki "yarı açık" bağlantıları temizler.
- **Silinirse:** `/api/v1/ws` çalışmaz, canlı bildirimler durur. REST API etkilenmez.
- **Benzetme:** Bir **radyo istasyonu**. `Hub` = yayın kontrol odası, `client` = her bir dinleyicinin alıcısı, `event_bridge` = haber ajansından gelen bültenleri anonsa çeviren editör.

#### 📁 `internal/infrastructure/gateway/interceptors/`

Gateway boru hattına takılan eklentiler:

| Dosya | Görev |
|---|---|
| `schema_validator.go` | Endpoint'te tanımlı JSON Schema'ya göre isteği doğrular |
| `pii_masker.go` | `password`, `ssn`, `credit_card` gibi alanları `***` yapar (iç içe nesnelerde de recursive) |
| `request_transformer.go` | Header/query/body dönüştürür; `{org_id}` gibi şablonları doldurur |
| `response_transformer.go` | Yanıt header/status/body dönüştürür |

- **Silinirse:** Şema doğrulama ve PII maskeleme kaybolur — güvenlik/uyumluluk (GDPR/KVKK) riski.
- **Benzetme:** Bir gazetenin **redaksiyon masası**. Yazı basılmadan önce format kontrolü ve sansür.

---

### 📁 `internal/gateway/` — Data Plane'in Beyni

> ⚠️ Dikkat: `internal/gateway/` (bu klasör), `internal/domain/gateway/` (arayüz tanımı) ve
> `internal/infrastructure/gateway/` (interceptor'lar) — **üçü farklı şeydir.**
> Bu klasör orkestrasyonu yapan katmandır.

**Bu klasör neden var?** Projenin en özgün özelliği olan "veritabanında tanımlı endpoint'lere gelen trafiği yönetme" işini yapmak için.

#### `pipeline.go` — Projenin En Kritik Dosyası
`Enforce` middleware'i şu adımları uygular:

```
1. Path atlanacaklardan mı? (/health, /api/v1/auth, /api/v1/ws) → geç
2. X-App-ID header'ı var mı? Yoksa → normal rotalara devret
3. DB'de bu (app, method, path, version) için endpoint kayıtlı mı? Yoksa → devret
4. Endpoint aktif mi? Değilse → 410 Gone
5. Policy'yi çek → gereken izin var mı? Yoksa → 403
6. Rate limit aşıldı mı? (Redis sayacı, 60 sn pencere) → 429
7. Context'e endpoint bilgilerini koy
8. Request interceptor zincirini çalıştır (şema doğrulama, PII maskeleme)
9. Backend'e yönlendir → yanıtı istemciye kopyala
```

- **Silinirse:** Dinamik endpoint sistemi tamamen ölür. Statik rotalar (org, app, user CRUD) çalışmaya devam eder.
- **Katman:** Middleware / Gateway orkestrasyonu
- **Benzetme:** **Havalimanı pasaport + güvenlik + kapı yönlendirme** kontrolünün tamamı.

> ⚠️ **Gözlem:** `pipeline.go` dosyasında adım 8'den sonra koşulsuz bir `return` var; ardından gelen "adım 9 — response interceptor" yorumu ve `responseWriter` yapısı şu an **ölü kod**. Yani `InterceptResponse` (dolayısıyla yanıtta PII maskeleme) pratikte hiç çalışmıyor. Kodu okurken bunu bilmek önemli.

#### `dynamic_handler.go` — Kodsuz CRUD Motoru
Endpoint için 3 strateji sırayla denenir:
1. **Kayıtlı handler** varsa onu kullan (`BackendRegistry`)
2. **URL** ise HTTP proxy yap (dış servise yönlendir)
3. Hiçbiri değilse → **jenerik veritabanı handler'ı**

3. strateji büyüleyici: `backend_service = "product-service"` → tablo adı `products` türetilir; `backend_action = "list"` → otomatik `SELECT ... WHERE organization_id = $1 AND app_id = $2` çalıştırılır. Yani **tek satır Go kodu yazmadan CRUD API'si** doğar.

- `row_to_json(t.*)` kullanımı akıllıca: kolon isimlerini önceden bilmeden sonucu JSON'a çevirir.
- **Her sorguda `organization_id` ve `app_id` filtresi var** → multi-tenant veri izolasyonu.

> ⚠️ **Gözlem:** Tablo adı `fmt.Sprintf` ile SQL'e gömülüyor (parametre değil — SQL'de tablo adı parametre olamaz). Tablo adı `backend_service` alanından türediği ve o alanı `endpoint:write` izni olan bir yönetici belirlediği için doğrudan istismar edilemez; yine de bir **allow-list** (izinli tablo listesi) eklemek doğru savunma katmanı olurdu. Öğrenirken bu ayrımı fark etmek önemli: "kullanıcı girdisi" ile "yönetici konfigürasyonu" farklı güven seviyeleridir.

#### `backend_handler.go`
- `BackendHandler` arayüzü + `BackendRegistry` (isim → handler eşlemesi).
- Özel iş mantığı gereken servisler için elle handler yazıp kaydedersin.
- **Benzetme:** Bir **santral operatörü**. "Muhasebe" dersin, doğru dahiliye bağlar.

#### `resolver.go`
- API key hash'inden org+app çözümü, subdomain'den org çözümü.
- Şu an `main.go`'da kullanılmıyor — gelecekteki API-key tabanlı erişim için hazır.

#### `handlers/example_handler.go`
- Kendi backend handler'ını nasıl yazacağını gösteren **canlı şablon**. Üretimde kullanılmaz, eğitim amaçlı.

---

### 📁 `internal/shared/` — Ortak Altyapı

**Neden var?** Her katmanın ihtiyaç duyduğu, iş mantığı içermeyen yardımcılar. **Uyarı:** Bu klasör kolayca "çöp kutusu"na dönüşebilir; buraya bir şey koymadan önce "gerçekten katmandan bağımsız mı?" diye sor.

| Paket | Görev | Silinirse |
|---|---|---|
| `config/` | Ortam değişkenlerinden konfigürasyon. Her ayarın makul varsayılanı var → `.env` olmadan da çalışır | Uygulama başlamaz |
| `logger/` | `slog` ile yapısal JSON log. Context'ten request_id/org_id/user_id çeker | Log kaybolur |
| `errors/` | Sentinel hatalar (`ErrNotFound` vb.) + HTTP kod eşlemesi | Hata yönetimi çöker |
| `response/` | JSON yanıt yardımcıları. **500'lerde iç hata mesajını gizler** | Yanıt üretilemez |
| `validator/` | `go-playground/validator` sarmalayıcısı; `DecodeAndValidate` tek adımda decode+validate | Girdi doğrulaması kaybolur |
| `pagination/` | Sayfalama. `MaxPerPage=100` ile DoS koruması. Go generics kullanıyor (`Result[T]`) | Liste endpoint'leri kırılır |
| `database/` | pgx bağlantı havuzu kurucusu | DB bağlantısı yok |
| `cache/` | Redis istemci kurucusu | Cache yok, RBAC yavaşlar |
| `events/` | `EventBus` arayüzü + in-process implementasyon + topic sabitleri | Olay sistemi çöker |
| `telemetry/` | OpenTelemetry + Prometheus metrik kurulumu | `/metrics` boşalır |
| `version/` | Sürüm ve servis adı sabitleri | Küçük etki |
| `middleware/` | Tüm HTTP middleware'leri | Güvenlik/gözlemlenebilirlik gider |

#### `shared/errors/errors.go` — Öğretici Desen
```go
var ErrNotFound = errors.New("resource not found")   // sentinel

type DomainError struct {
    Kind    error   // yukarıdaki sentinel'lerden biri
    Message string  // insan-okur mesaj
    Err     error   // debug için sarmalanan hata
}
func (e *DomainError) Unwrap() error { return e.Kind }  // errors.Is() çalışsın diye
```
Sonra `HTTPStatusCode(err)` bunu 404/409/403'e çevirir. **Sonuç:** İş katmanı HTTP kodu bilmeden hata döner, HTTP katmanı otomatik doğru kodu üretir. Katman ayrımının ders kitabı örneği.

#### `shared/middleware/` — Dosya Dosya

| Dosya | Görev | Benzetme |
|---|---|---|
| `request_id.go` | Her isteğe UUID atar (varsa header'dakini kullanır) | Kargo takip numarası |
| `logging.go` | Method, path, status, süre loglar | Ziyaretçi defteri |
| `recoverer.go` | Panik yakalar, stack trace loglar, 500 döner | Emniyet ağı |
| `body_limit.go` | Body boyutunu sınırlar (varsayılan 1MB) | Bagaj ağırlık limiti |
| `cors.go` | Tarayıcı origin politikası. **`*` varsa credentials otomatik kapatılır** (doğru güvenlik davranışı) | Vize politikası |
| `auth.go` | JWT doğrular, claim'leri context'e koyar; `RequirePermission` RBAC kontrolü | Pasaport kontrolü |
| `auth_ws.go` | WebSocket için token'ı `?token=` query'sinden de okur (tarayıcılar WS'te header koyamaz) | Yan kapı girişi |
| `tenant.go` | Organizasyonu çözer: `X-Organization-ID` header → JWT claim → subdomain sırasıyla | Hangi şubedesin? |
| `audit.go` | Her isteği audit log'a yazar (asenkron, best-effort) | Güvenlik kamerası |

> ⚠️ **Gözlem:** `audit.go` yazılmış ama `router.go`'da **hiç kullanılmıyor**. Yani otomatik denetim kaydı şu an aktif değil; `audit_logs` tablosuna veri yazan tek yol dışarıdan gelecek başka bir mekanizma. Ayrıca middleware'in `go func(){}` içinde `r.Context()` kullanması riskli — istek bitince context iptal olur ve yazma başarısız olabilir.

---

### 📁 `deployments/`

**Neden var?** Uygulamayı nerede/nasıl çalıştıracağını tarif eder. Uygulama kodundan ayrılır ki farklı ortamlar (dev/prod) farklı dosyalarla yönetilebilsin.

#### `Dockerfile`
- **Multi-stage build:** 1. aşamada `golang:1.26.4-alpine` ile derlenir, 2. aşamada sadece `alpine:3.24` üzerine binary kopyalanır.
- Sonuç: ~1GB yerine ~20MB imaj. Daha az saldırı yüzeyi.
- **`USER appuser`** — root olarak çalışmaz. Container güvenliğinin temel kuralı.
- **Silinirse:** Docker imajı üretilemez, üretime çıkamazsın.

#### `docker-compose.yml`
- Yerel geliştirme yığını: Postgres 16, Redis 7, Kafka (KRaft modu, Zookeeper'sız), Kafka UI.
- **Portlar `127.0.0.1`'e bağlı** — varsayılan olarak dış ağa açılmaz. Bilinçli güvenlik tercihi.
- `healthcheck` tanımları var → `dev.sh` servislerin hazır olmasını bekleyebiliyor.
- **Benzetme:** Bir **prova sahnesi**. Gerçek konser salonunun (prod) küçük ölçekli kopyası.

---

### 📁 `scripts/`

| Dosya | Görev |
|---|---|
| `migrate.sh` | Migration'ları çalıştırır. `goose` yoksa `docker exec psql` ile SQL'i doğrudan basar |
| `seed.go` | Başlangıç verisi: `admin` (izin: `*`), `member`, `viewer` rolleri |
| `test.sh` | Test koşturucu |
| `lint.sh` | `golangci-lint` çalıştırıcı |

- **`seed.go` neden ayrı bir `main` paketi?** `go run scripts/seed.go` ile bağımsız çalışsın diye. Sunucu binary'sine dahil olmaz.
- **Silinirse:** Manuel işlem gerekir; uygulama çalışmaya devam eder.
- **Benzetme:** Bir evin **anahtar teslim öncesi hazırlık listesi**.

---

### 📁 `docs/`, `postman/`, `.github/`, `.cursor/`

| Klasör | Amaç |
|---|---|
| `docs/WEBSOCKET.md` | WebSocket protokol dokümantasyonu |
| `postman/` | Hazır API test koleksiyonu + ortam değişkenleri. `POSTMAN_COLLECTION_GUIDE.md` ile birlikte |
| `.github/` | Issue/PR şablonları, banner görseli |
| `.cursor/`, `.cursor-plugin/` | Cursor IDE için AI kuralları: kodlama konvansiyonları, "yeni endpoint nasıl eklenir" şablonları |
| `internal/gateway/BACKEND_HANDLERS.md` | Kendi backend handler'ını yazma kılavuzu |

**`.cursor/rules/` özellikle değerli** — projenin kendi mimari konvansiyonlarını (`create-usecase.mdc`, `create-handler.mdc`, `error-handling.mdc`) yazılı hale getirmiş. Kod yazmadan önce oraya bakmakta fayda var.

---

### 📄 Kök Dizin Dosyaları

| Dosya | Amaç | Silinirse |
|---|---|---|
| `go.mod` | Modül adı + Go sürümü + doğrudan bağımlılıklar | Proje derlenmez |
| `go.sum` | Bağımlılıkların kriptografik hash'leri (tedarik zinciri güvenliği) | Doğrulama kaybolur |
| `Makefile` | Standart komutlar: `build`, `run`, `test`, `migrate`, `docker-up`, `seed` | Komutları elle yazarsın |
| `dev.sh` | Tek komutla tam geliştirme ortamı: Docker + migration + hot-reload | Manuel kurulum |
| `.air.toml` | Hot-reload ayarları. `.go` dosyası kaydedince ~3 sn'de yeniden derler | Her değişiklikte elle restart |
| `.gitignore` | `bin/`, `tmp/`, `.env` gibi commit'lenmemesi gerekenler | **Sırlar repoya sızabilir** |
| `README.md` | Ana dokümantasyon (22KB — kapsamlı) | Yeni geliştirici kaybolur |
| `SECURITY.md` | Güvenlik politikası ve açık bildirim süreci | — |
| `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`, `LICENSE` | Açık kaynak proje hijyeni (AGPL-3.0) | — |
| `TEST_RESULTS.md` | Test sonuç raporu | — |

---

## 5. Test Stratejisi

Testler kaynak dosyanın **yanında** durur (`_test.go` soneki) — Go konvansiyonu.

| Test dosyası | Neyi test ediyor |
|---|---|
| `shared/config/config_test.go` | Env okuma, varsayılanlar, DSN üretimi |
| `shared/errors/errors_test.go` | Hata → HTTP kod eşlemesi |
| `shared/pagination/pagination_test.go` | Sınır değerleri (0, negatif, 1000) |
| `shared/middleware/*_test.go` | CORS, body limit, WS token çıkarma |
| `infrastructure/auth/jwt_service_test.go` | Token üretme/doğrulama, bozuk token, imza saldırısı |
| `infrastructure/auth/rbac_service_test.go` | Joker izin eşleme mantığı |
| `application/tenant/usecase/create_workspace_test.go` | **Sahte (mock) repository ile use case testi** — bu, Clean Architecture'ın somut kazancı |
| `domain/realtime/model/room_test.go` | Oda anahtarı üretme/çözme |
| `gateway/dynamic_handler_test.go` | Tablo adı türetme, URL kontrolü |

`make test` → `go test -v -race -count=1 ./...` (`-race` = yarış koşulu dedektörü, WebSocket/Hub için kritik)

---

## 6. Gerçek Hayat Benzetmeleri — Toplu Tablo

| Bileşen | Gerçek hayat karşılığı |
|---|---|
| `cmd/server/main.go` | Orkestra şefi |
| `router.go` | Bina kat planı + tabelalar |
| `middleware/` | Giriş kapısındaki kontrol noktaları zinciri |
| `handler/` | Danışma masası |
| `usecase/` | İş akışı kontrol listesi |
| `domain/model/` | Şirketin sözlüğü / terminolojisi |
| `domain/repository/` (interface) | Duvardaki priz standardı |
| `infrastructure/postgres/` | Arşiv görevlisi |
| `infrastructure/kafka/` | Postane |
| `infrastructure/websocket/` | Radyo istasyonu |
| `infrastructure/auth/` | Pasaport ofisi |
| `gateway/pipeline.go` | Havalimanı güvenlik hattı |
| `gateway/dynamic_handler.go` | Sipariş üzerine üretim tezgâhı |
| `migrations/` | Tadilat ruhsatları dosyası |
| `shared/errors/` | Ortak hata kodu sözlüğü |
| `deployments/` | Prova sahnesi |

---

## 7. Önerilen Öğrenme Sırası

Bu projeyi anlamak için dosyaları **bu sırayla** oku:

1. `cmd/server/main.go` — özellikle `buildDependencies()`. Tüm haritayı görürsün.
2. `internal/infrastructure/http/router/router.go` — URL → kod eşlemesi.
3. **Dikey bir dilim seç ve baştan sona takip et.** Örnek: kullanıcı kaydı
   - `handler/iam/handler.go` → `Register()`
   - `application/iam/dto/user_dto.go` → `RegisterRequest`
   - `application/iam/usecase/register.go` → `Execute()`
   - `domain/iam/model/user.go` → `User`
   - `domain/iam/repository/user_repository.go` → arayüz
   - `infrastructure/postgres/iam/user_repository.go` → SQL
4. `shared/middleware/auth.go` + `infrastructure/auth/jwt_service.go` — güvenlik zinciri.
5. `internal/gateway/pipeline.go` — projenin özgün fikri.
6. `shared/events/` + `infrastructure/websocket/event_bridge.go` — olay akışı.

**Pratik alıştırma:** Bir "Notification" bounded context'i eklemeyi dene. Aynı 6 dosyayı (model, repository interface, postgres repo, dto, usecase, handler) yazıp `main.go` ve `router.go`'ya bağlaman gerekecek. Bu, deseni ezberlemenin en hızlı yolu.

---

## 8. Mimari Değerlendirme — Güçlü ve Zayıf Yanlar

### Güçlü
- Katman ayrımı gerçekten uygulanmış; domain katmanı temiz
- Hata çevirisi deseni (pgx → domain error → HTTP kod) örnek niteliğinde
- Güvenlik detaylarına özen: `json:"-"` ile sır gizleme, JWT algoritma kontrolü, API key hash'leme, non-root container, loopback port binding
- `EventBus` soyutlaması sayesinde Kafka isteğe bağlı
- Altyapı yokken bile ayağa kalkabilme (graceful degradation)
- Veritabanı-güdümlü endpoint tanımı gerçekten yaratıcı bir fikir

### Dikkat Edilecekler
- `context.WithValue` string anahtarlarla kullanılmış (`"endpoint_schema"`, `"org_id"`) — Go'da bu çakışma riskli, tipli anahtar (`type ctxKey string`) tercih edilmeli. `shared/middleware` doğru yapıyor ama `gateway/pipeline.go` yapmıyor.
- `pipeline.go`'da response interceptor'lar ölü kod (erken `return`)
- `audit.go` middleware'i yazılmış ama bağlanmamış
- Application katmanı `shared/middleware`'i import ediyor (küçük katman ihlali)
- Dinamik SQL'de tablo adı için allow-list yok
- Rate limit sabit 60 sn pencere; sliding window değil (pencere sınırında 2x istek geçebilir)

---

*Bu rehber `masterfabric-go` v0.0.1 kod tabanı üzerinden hazırlanmıştır.*
