package main

import "github.com/masterfabric-go/masterfabric/internal/domain/triage/model"

// customerSeed is one demo customer.
type customerSeed struct {
	Email    string
	FullName string
	Company  string
}

var demoCustomers = []customerSeed{
	{"ayse.demir@modabutik.com", "Ayşe Demir", "Moda Butik"},
	{"mehmet.yilmaz@acmeticaret.com", "Mehmet Yılmaz", "Acme Ticaret"},
	{"zeynep.kaya@teknomarket.com", "Zeynep Kaya", "Tekno Market"},
	{"emre.sahin@evdekor.com", "Emre Şahin", "Ev Dekor"},
	{"fatma.celik@organikgida.com", "Fatma Çelik", "Organik Gıda"},
	{"burak.arslan@sporcenter.com", "Burak Arslan", "Spor Center"},
	{"elif.dogan@kitapdunyasi.com", "Elif Doğan", "Kitap Dünyası"},
	{"can.ozturk@otoyedek.com", "Can Öztürk", "Oto Yedek Parça"},
	{"selin.aydin@kozmetikhouse.com", "Selin Aydın", "Kozmetik House"},
	{"murat.kilic@yapimarket.com", "Murat Kılıç", "Yapı Market"},
	{"deniz.polat@bebekshop.com", "Deniz Polat", "Bebek Shop"},
	{"gizem.avci@petdunyasi.com", "Gizem Avcı", "Pet Dünyası"},
	{"okan.tas@mobilyaevi.com", "Okan Taş", "Mobilya Evi"},
	{"nur.erdogan@takiatolyesi.com", "Nur Erdoğan", "Takı Atölyesi"},
}

// ticketSeed is one demo ticket. Category records what the text is written to
// trigger; the stub classifier still decides on its own, and the seed verifies
// the outcome rather than forcing it.
type ticketSeed struct {
	Subject  string
	Body     string
	Category model.Category
}

// ambiguousCategory marks tickets written WITHOUT classifier keywords. They
// exercise the low-confidence path: the stub finds nothing, confidence sits at
// the floor and needs_human_review turns true.
const ambiguousCategory model.Category = ""

// demoTickets is the ticket corpus. Volume is weighted the way a Turkish B2B
// SaaS help desk actually looks: integration and platform faults dominate,
// money operations and how-to questions follow.
var demoTickets = []ticketSeed{
	// ── integration (16) ─────────────────────────────────────────────────────
	{"Trendyol entegrasyonu ürünleri aktarmıyor", "Trendyol pazaryeri entegrasyonumuz dün akşamdan beri ürünleri aktarmıyor, senkronizasyon sürekli hata veriyor.", model.CategoryIntegration},
	{"Hepsiburada stok senkronizasyonu çalışmıyor", "Hepsiburada entegrasyonunda stok adetleri güncellenmiyor, senkron işlemi yarım kalıyor.", model.CategoryIntegration},
	{"N11 siparişleri panele düşmüyor", "N11 pazaryeri entegrasyonu siparişleri çekmiyor, API tarafında hata dönüyor.", model.CategoryIntegration},
	{"Kargo etiketi basmıyor", "Yurtiçi kargo entegrasyonunda etiket oluşturulmuyor, kargo servisi hata veriyor ve sevkiyat yapamıyoruz.", model.CategoryIntegration},
	{"Logo muhasebe entegrasyonu aktarım yapmıyor", "Logo ERP entegrasyonu kayıtları muhasebeye aktarmıyor, dün geceden beri senkron yok.", model.CategoryIntegration},
	{"Mikro ERP bağlantısı koptu", "Mikro muhasebe entegrasyonunun bağlantısı koptu, veri aktarımı tamamen durdu.", model.CategoryIntegration},
	{"Netsis entegrasyonunda stok kodları eşleşmiyor", "Netsis ERP entegrasyonunda ürün kodları eşleşmediği için aktarım hata veriyor.", model.CategoryIntegration},
	{"Sanal POS entegrasyonu hata veriyor", "Sanal POS entegrasyonumuz ödeme adımında hata dönüyor, banka tarafında sorun görünmüyor.", model.CategoryIntegration},
	{"Webhook bildirimleri gelmiyor", "Sipariş webhook bildirimleri iki gündür sunucumuza ulaşmıyor, entegrasyon loglarında kayıt yok.", model.CategoryIntegration},
	{"API anahtarımız çalışmıyor", "Yeni oluşturduğumuz API anahtarı ile istek attığımızda entegrasyon yetki hatası dönüyor.", model.CategoryIntegration},
	{"SDK güncellemesi sonrası senkron bozuldu", "SDK sürümünü güncelledikten sonra pazaryeri senkronizasyonu hata vermeye başladı.", model.CategoryIntegration},
	{"Trendyol komisyon bilgisi aktarılmıyor", "Trendyol entegrasyonunda komisyon alanı boş geliyor, aktarım eksik yapılıyor.", model.CategoryIntegration},
	{"Pazaryeri fiyat güncellemesi gecikiyor", "Pazaryeri entegrasyonunda fiyat güncellemeleri saatler sonra yansıyor, senkron çok yavaş.", model.CategoryIntegration},
	{"Aras kargo takip numarası dönmüyor", "Aras kargo entegrasyonu gönderi oluşturuyor ama takip numarası aktarmıyor.", model.CategoryIntegration},
	{"Hepsiburada iadeleri aktarılmıyor", "Hepsiburada entegrasyonunda iade siparişleri panele aktarılmıyor, senkron atlıyor.", model.CategoryIntegration},
	{"API rate limit hatası alıyoruz", "Entegrasyon servisimiz API üzerinden istek atarken sürekli limit hatası alıyor.", model.CategoryIntegration},

	// ── technical_issue (16) ─────────────────────────────────────────────────
	{"Panel açılmıyor, 500 hatası alıyoruz", "Sabahtan beri yönetim paneli açılmıyor, ekranda 500 hatası görünüyor.", model.CategoryTechnicalIssue},
	{"Raporlar yüklenmiyor", "Satış raporları sayfası yüklenmiyor, uzun süre bekledikten sonra hata veriyor.", model.CategoryTechnicalIssue},
	{"Site çok yavaş çalışıyor", "Panel bugün aşırı yavaş, sayfalar geç açılıyor ve zaman zaman hata veriyor.", model.CategoryTechnicalIssue},
	{"Sipariş listesi yüklenmiyor", "Sipariş listesi ekranı yüklenmiyor, sayfa boş kalıyor ve konsolda hata var.", model.CategoryTechnicalIssue},
	{"Ürün ekleme sayfası hata veriyor", "Yeni ürün eklerken kaydet butonuna basınca hata alıyoruz, kayıt oluşmuyor.", model.CategoryTechnicalIssue},
	{"Dashboard grafikleri boş geliyor", "Ana ekrandaki grafikler yüklenmiyor, veri gelmiyor ve hata mesajı çıkıyor.", model.CategoryTechnicalIssue},
	{"Arama sonuçları çok yavaş dönüyor", "Ürün araması aşırı yavaş, bazen hiç sonuç dönmeden hata veriyor.", model.CategoryTechnicalIssue},
	{"Excel dışa aktarma hata veriyor", "Rapor Excel olarak indirilmek istendiğinde işlem hata ile sonuçlanıyor.", model.CategoryTechnicalIssue},
	{"Görsel yükleme başarısız oluyor", "Ürün görseli yüklerken işlem hata veriyor, dosya bir türlü yüklenmiyor.", model.CategoryTechnicalIssue},
	{"Site kapalı, müşterilerim giremiyor", "Sitemiz tamamen kapalı, müşterilerim etkileniyor ve satış yapamıyorum.", model.CategoryTechnicalIssue},
	{"Filtreleme çalışmıyor", "Sipariş ekranındaki filtreler çalışmıyor, uygulayınca hata veriyor.", model.CategoryTechnicalIssue},
	{"Toplu güncelleme ekranı açılmıyor", "Toplu fiyat güncelleme ekranı açılmıyor, sürekli hata veriyor.", model.CategoryTechnicalIssue},
	{"Bildirimler gelmiyor", "Panel içi bildirimler görünmüyor, sayfa yüklenmiyor ve hata düşüyor.", model.CategoryTechnicalIssue},
	{"Mobil uygulamada beyaz ekran", "Mobil uygulama açılışta beyaz ekranda kalıyor ve hata veriyor.", model.CategoryTechnicalIssue},
	{"Yazdırma önizlemesi yüklenmiyor", "Fiş yazdırma önizlemesi yüklenmiyor, ekran boş kalıyor ve hata çıkıyor.", model.CategoryTechnicalIssue},
	{"Panel çok yavaş, zaman aşımı alıyoruz", "Panelde işlemler çok yavaş ilerliyor, sık sık timeout hatası alıyoruz.", model.CategoryTechnicalIssue},

	// ── payment_ops (10) ─────────────────────────────────────────────────────
	{"Dünkü hakedişim hesabıma geçmedi", "Dün kapanan hakediş tutarı hesabıma geçmedi, ödeme alamıyorum.", model.CategoryPaymentOps},
	{"İade işlemi müşteriye yansımadı", "Onayladığımız iade tutarı müşterinin kartına yansımadı, iade süreci takılı.", model.CategoryPaymentOps},
	{"Chargeback bildirimi aldık", "Bir işlem için chargeback bildirimi geldi, ters ibraz süreci hakkında bilgi almak istiyoruz.", model.CategoryPaymentOps},
	{"Settlement raporu eksik görünüyor", "Bu haftanın settlement raporunda bazı işlemler eksik görünüyor.", model.CategoryPaymentOps},
	{"Ödeme alamıyorum", "Tahsilat yapamıyorum, gelen ödemeler hesabıma geçmedi ve hakediş görünmüyor.", model.CategoryPaymentOps},
	{"Hakediş tutarı eksik hesaplanmış", "Bu ayki hakediş tutarı beklediğimizden düşük, mutabakat farkı var.", model.CategoryPaymentOps},
	{"İade tutarı hatalı hesaplandı", "Kısmi iade işleminde iade tutarı yanlış hesaplandı, para eksik döndü.", model.CategoryPaymentOps},
	{"Mutabakat raporunda fark var", "Mutabakat dosyası ile panel arasındaki tutarlar uyuşmuyor, işlem eksik görünüyor.", model.CategoryPaymentOps},
	{"Para transferi gecikti", "Hakediş ödemesi normalde iki günde geliyordu, bu kez para yatmadı.", model.CategoryPaymentOps},
	{"İşlem eksik görünüyor", "Panelde dünkü işlemlerden biri eksik görünüyor, settlement tarafında da yok.", model.CategoryPaymentOps},

	// ── how_to (9) ───────────────────────────────────────────────────────────
	{"Kampanya nasıl oluşturulur?", "Sezon indirimi için kampanya nasıl oluşturulur, adımları öğrenmek istiyoruz.", model.CategoryHowTo},
	{"Toplu ürün yükleme nasıl yapılır?", "Elimizdeki listeyi toplu olarak nasıl yükleriz, doküman var mı?", model.CategoryHowTo},
	{"Kullanım kılavuzuna nereden ulaşırım?", "Panelin kullanım kılavuzu veya dokümantasyonu nerede bulunuyor?", model.CategoryHowTo},
	{"Raporları nasıl filtrelerim?", "Rapor ekranında tarih bazlı filtrelemeyi nasıl yaparım, kılavuz var mı?", model.CategoryHowTo},
	{"Eğitim videoları var mı?", "Yeni işe başlayan ekip için eğitim içerikleri veya doküman mevcut mu?", model.CategoryHowTo},
	{"Bildirim ayarlarını nasıl konfigüre ederim?", "Bildirim tercihlerini nasıl ayarlarım, konfigüre etmek için doküman var mı?", model.CategoryHowTo},
	{"Stok uyarı seviyesi nasıl ayarlanır?", "Kritik stok uyarısını nasıl ayarlamak gerekiyor, öğrenmek istiyoruz.", model.CategoryHowTo},
	{"Raporlama modülünü öğrenmek istiyoruz", "Raporlama modülü için eğitim veya kılavuz talep ediyoruz.", model.CategoryHowTo},
	{"Panel ayarlarını nasıl değiştiririm?", "Genel panel ayarlarını nasıl değiştirebilirim, dokümanda göremedim.", model.CategoryHowTo},

	// ── billing (3) — bu org'da karşılık gelen departman YOK ──────────────────
	{"Şubat ayı faturamı göremiyorum", "Şubat ayına ait faturamız panelde görünmüyor, fatura talep ediyoruz.", model.CategoryBilling},
	{"Abonelik paketimizi yükseltmek istiyoruz", "Mevcut abonelik paketimizi bir üst pakete yükseltmek istiyoruz.", model.CategoryBilling},
	{"Komisyon oranımız yanlış", "Sözleşmede belirtilen komisyon oranı ile faturaya yansıyan oran farklı.", model.CategoryBilling},

	// ── onboarding (3) ───────────────────────────────────────────────────────
	{"Kurulum sürecinde takıldık", "Kurulum adımlarında ilerleyemiyoruz, setup sırasında destek almak istiyoruz.", model.CategoryOnboarding},
	{"Veri göçü ne kadar sürer?", "Eski sistemden veri göçü süreci ne kadar sürüyor, migration planı nedir?", model.CategoryOnboarding},
	{"Canlıya çıkış için neler gerekli?", "Canlıya geçiş öncesi aktivasyon için hangi adımları tamamlamamız gerekiyor?", model.CategoryOnboarding},

	// ── account_access (3) ───────────────────────────────────────────────────
	{"Şifremi sıfırlayamıyorum", "Şifre sıfırlama maili gelmiyor, panele giriş yapamıyorum.", model.CategoryAccountAccess},
	{"Yeni kullanıcı ekleyemiyoruz", "Ekibe yeni kullanıcı ekle dediğimizde yetki hatası alıyoruz, rol atayamıyoruz.", model.CategoryAccountAccess},
	{"Panele erişim iznimiz yok", "Muhasebe ekibinin panele erişimi yok, yetki tanımlaması yapılamıyor.", model.CategoryAccountAccess},

	// ── feature_request (3) ──────────────────────────────────────────────────
	{"Toplu indirim özelliği talebi", "Ürün gruplarına toplu indirim uygulayabileceğimiz bir özellik talebi iletmek istiyoruz.", model.CategoryFeatureRequest},
	{"Yol haritasında çoklu depo var mı?", "Çoklu depo yönetimi yol haritasında mı, roadmap paylaşabilir misiniz?", model.CategoryFeatureRequest},
	{"Çoklu para birimi destekliyor musunuz?", "Farklı para birimlerinde satış yapmayı destekliyor musunuz, öneri olarak iletiyoruz.", model.CategoryFeatureRequest},

	// ── sales (3) ────────────────────────────────────────────────────────────
	{"Ek modül için teklif istiyoruz", "Raporlama ek modülünü satın almak istiyoruz, fiyat teklifi rica ederiz.", model.CategorySales},
	{"Demo talebi", "Kurumsal sürüm için demo talebimiz var, uygun bir zamanda görüşebilir miyiz?", model.CategorySales},
	{"Fiyat listesi paylaşabilir misiniz?", "Yeni paket seçenekleri için güncel fiyat listesi ve teklif rica ediyoruz.", model.CategorySales},

	// ── compliance (3) ───────────────────────────────────────────────────────
	{"KVKK kapsamında veri silme talebi", "KVKK kapsamında şirketimize ait verilerin imha edilmesini talep ediyoruz.", model.CategoryCompliance},
	{"Aydınlatma metni ve sözleşme talebi", "Aydınlatma metni ile güncel sözleşme örneğini paylaşmanızı rica ederiz.", model.CategoryCompliance},
	{"Denetim için belge talebi", "Yıllık denetim sürecimiz için gerekli belge talebimiz bulunuyor.", model.CategoryCompliance},

	// ── belirsiz (7) — anahtar kelime yok, düşük güven senaryosu ─────────────
	{"Bir konuda yardım rica ediyorum", "Merhaba, dün görüştüğümüz konuyla ilgili tarafınızdan dönüş bekliyoruz.", ambiguousCategory},
	{"Kısa bir sorum olacak", "Geçen hafta ilettiğimiz talep hakkında güncel durum nedir acaba?", ambiguousCategory},
	{"Geri dönüş bekliyoruz", "Konuyla ilgili ekibinizden henüz bir yanıt alamadık, teşekkürler.", ambiguousCategory},
	{"Görüşmemizin devamı", "Salı günkü toplantıda konuştuklarımızı ilerletmek istiyoruz.", ambiguousCategory},
	{"Ekteki konu hakkında", "Daha önce ilettiğimiz konuyla ilgili bir gelişme var mı?", ambiguousCategory},
	{"Ufak bir talebimiz var", "Uygun olduğunuzda bizimle iletişime geçebilir misiniz?", ambiguousCategory},
	{"Durum güncellemesi", "Süreç hangi aşamada, bilgilendirme yapabilir misiniz?", ambiguousCategory},
}

// agentReplies feed the multi-message threads.
var agentReplies = []string{
	"Merhaba, talebinizi aldık. İlgili ekip inceliyor, en kısa sürede dönüş yapacağız.",
	"Konuyu teknik ekibe ilettik. Loglarda ilgili kaydı bulduk, üzerinde çalışıyoruz.",
	"Sorunu tespit ettik, bir düzeltme hazırlanıyor. Yayına alındığında bilgilendireceğiz.",
	"Kontrollerimizi tamamladık, işlemi tarafımızdan yeniden başlattık. Teyit eder misiniz?",
	"Bilgi için teşekkürler, süreci güncelledik ve takibe aldık.",
}

var customerFollowUps = []string{
	"Teşekkürler, ne zaman çözüleceği konusunda bilgi verebilir misiniz?",
	"Sorun devam ediyor, ekran görüntüsünü paylaşıyorum.",
	"Bugün tekrar denedik ancak aynı durumla karşılaştık.",
	"Anladım, bekliyoruz. Acil olduğunu tekrar belirtmek isterim.",
}

var internalNotes = []string{
	"Dahili not: müşteri kritik hesap, SLA takibi yapılmalı.",
	"Dahili not: benzer kayıt geçen hafta da açılmıştı, tekrar eden sorun.",
	"Dahili not: entegrasyon ekibine eskale edildi.",
}
