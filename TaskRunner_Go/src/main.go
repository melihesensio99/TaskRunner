/*
=================================================
*
* Açıklama: Bu ödevde bağımlılıkları çözerek 
* task'ları paralel çalıştıran küçük bir dil 
* tasarladım ve yorumlayıcısını yazdım.
=================================================
*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Komutları tuttuğum yapı. exec veya print olabilir.
type Komut struct {
	KomutTuru string 
	Arguman   string // Ekrana basılacak yazı veya işletim sistemi komutu
}

// Her bir task'in bilgilerini burada tutuyorum.
type Gorev struct {
	GorevAdi   string
	Bagimliliklar []string
	KomutListesi  []Komut
}

// --- Sözcük Analizi (Lexer) Kısmı ---

type JetonTipi int

const (
	JetonBitir JetonTipi = iota
	JetonKelime
	JetonMetin
	JetonTask
	JetonDepends
	JetonExec
	JetonPrint
	JetonSolSüslü 
	JetonSagSüslü 
	JetonNoktaliVirgül
	JetonVirgül
)

type Jeton struct {
	Tipi   JetonTipi
	Degeri string
	SatirNo int
}

type SözcükOkuyucu struct {
	kaynakKod     string
	suAnkiKonum   int
	siradakiKonum int
	karakter      byte
	satirSayaci   int
}

// Dosyayı okumaya başlamak için ilk hazırlık
func OkuyucuOlustur(kod string) *SözcükOkuyucu {
	okuyucu := &SözcükOkuyucu{kaynakKod: kod, satirSayaci: 1}
	okuyucu.karakterOku()
	return okuyucu
}

// Bir sonraki karaktere geçiş fonksiyonu
func (o *SözcükOkuyucu) karakterOku() {
	if o.siradakiKonum >= len(o.kaynakKod) {
		o.karakter = 0 // dosya sonu
	} else {
		o.karakter = o.kaynakKod[o.siradakiKonum]
	}
	// Alt satıra geçtiysek satır sayacını artıralım, hatada lazım olacak
	if o.karakter == '\n' {
		o.satirSayaci++
	}
	o.suAnkiKonum = o.siradakiKonum
	o.siradakiKonum++
}

// Boşlukları atlayan yardımcı fonksiyon
func (o *SözcükOkuyucu) bosluklariAtla() {
	for o.karakter == ' ' || o.karakter == '\t' || o.karakter == '\n' || o.karakter == '\r' {
		o.karakterOku()
	}
}

// Her çağrıldığında sıradaki jetonu (token) bulup döndürür
func (o *SözcükOkuyucu) SiradakiJetonuGetir() Jeton {
	o.bosluklariAtla()
	var j Jeton
	j.SatirNo = o.satirSayaci

	switch o.karakter {
	case '{':
		j = Jeton{Tipi: JetonSolSüslü, Degeri: "{"}
	case '}':
		j = Jeton{Tipi: JetonSagSüslü, Degeri: "}"}
	case ';':
		j = Jeton{Tipi: JetonNoktaliVirgül, Degeri: ";"}
	case ',':
		j = Jeton{Tipi: JetonVirgül, Degeri: ","}
	case '"':
		j.Tipi = JetonMetin
		j.Degeri = o.metinOku()
	case 0:
		j.Tipi = JetonBitir
		j.Degeri = ""
	default:
		// Harf geliyorsa bir kelime okuyoruz demektir (task, depends vs.)
		if harfMi(o.karakter) {
			j.Degeri = o.kelimeOku()
			j.Tipi = kelimeTipiniBul(j.Degeri)
			return j
		}
	}
	o.karakterOku()
	return j
}

// Tırnak içindeki yazıları almak için
func (o *SözcükOkuyucu) metinOku() string {
	baslangic := o.suAnkiKonum + 1
	for {
		o.karakterOku()
		if o.karakter == '"' || o.karakter == 0 {
			break
		}
	}
	return o.kaynakKod[baslangic:o.suAnkiKonum]
}

// Harf ve rakamlardan oluşan kelimeleri yakalamak için
func (o *SözcükOkuyucu) kelimeOku() string {
	baslangic := o.suAnkiKonum
	for harfMi(o.karakter) || rakamMi(o.karakter) || o.karakter == '_' {
		o.karakterOku()
	}
	return o.kaynakKod[baslangic:o.suAnkiKonum]
}

func harfMi(k byte) bool {
	return ('a' <= k && k <= 'z') || ('A' <= k && k <= 'Z') || k == '_'
}

func rakamMi(k byte) bool {
	return '0' <= k && k <= '9'
}

func kelimeTipiniBul(kelime string) JetonTipi {
	switch kelime {
	case "task":
		return JetonTask
	case "depends":
		return JetonDepends
	case "exec":
		return JetonExec
	case "print":
		return JetonPrint
	default:
		return JetonKelime
	}
}

// --- Ayrıştırıcı (Parser) Kısmı ---

type Ayristirici struct {
	okuyucu       *SözcükOkuyucu
	suAnkiJeton   Jeton
	siradakiJeton Jeton
	hataListesi   []string
}

func AyristiriciOlustur(o *SözcükOkuyucu) *Ayristirici {
	ayristirici := &Ayristirici{okuyucu: o}
	// İlk iki jetonu önden okuyalım
	ayristirici.jetonIlerlet()
	ayristirici.jetonIlerlet()
	return ayristirici
}

func (a *Ayristirici) jetonIlerlet() {
	a.suAnkiJeton = a.siradakiJeton
	a.siradakiJeton = a.okuyucu.SiradakiJetonuGetir()
}

// Kaynak dosyadaki tüm task'leri ayıklayıp dizi olarak döner
func (a *Ayristirici) ProgramiAyristir() []Gorev {
	var gorevler []Gorev
	for a.suAnkiJeton.Tipi != JetonBitir {
		yeniGorev := a.goreviAyristir()
		if yeniGorev != nil {
			gorevler = append(gorevler, *yeniGorev)
		} else {
			a.jetonIlerlet()
		}
	}
	return gorevler
}

func (a *Ayristirici) goreviAyristir() *Gorev {
	if a.suAnkiJeton.Tipi != JetonTask {
		a.hataListesi = append(a.hataListesi, fmt.Sprintf("Satır %d: Kod 'task' ile başlamalıydı ama '%s' buldum.", a.suAnkiJeton.SatirNo, a.suAnkiJeton.Degeri))
		return nil
	}
	a.jetonIlerlet()

	if a.suAnkiJeton.Tipi != JetonKelime {
		a.hataListesi = append(a.hataListesi, fmt.Sprintf("Satır %d: Task'a bir isim verilmeliydi.", a.suAnkiJeton.SatirNo))
		return nil
	}
	
	olusturulanGorev := &Gorev{GorevAdi: a.suAnkiJeton.Degeri}
	a.jetonIlerlet()

	// Eğer depends yazıyorsa bağımlılıkları listeye atalım
	if a.suAnkiJeton.Tipi == JetonDepends {
		a.jetonIlerlet()
		for {
			if a.suAnkiJeton.Tipi != JetonKelime {
				a.hataListesi = append(a.hataListesi, fmt.Sprintf("Satır %d: Bağımlılık ismi bekliyordum.", a.suAnkiJeton.SatirNo))
				return nil
			}
			olusturulanGorev.Bagimliliklar = append(olusturulanGorev.Bagimliliklar, a.suAnkiJeton.Degeri)
			a.jetonIlerlet()
			
			// Virgül varsa başka bağımlılık da var demektir
			if a.suAnkiJeton.Tipi == JetonVirgül {
				a.jetonIlerlet()
			} else {
				break
			}
		}
	}

	if a.suAnkiJeton.Tipi != JetonSolSüslü {
		a.hataListesi = append(a.hataListesi, fmt.Sprintf("Satır %d: Task bloğu '{' ile başlamalı.", a.suAnkiJeton.SatirNo))
		return nil
	}
	a.jetonIlerlet()

	// Süslü parantez kapanana kadar içerideki komutları okuyoruz
	for a.suAnkiJeton.Tipi != JetonSagSüslü && a.suAnkiJeton.Tipi != JetonBitir {
		k := a.komutuAyristir()
		if k != nil {
			olusturulanGorev.KomutListesi = append(olusturulanGorev.KomutListesi, *k)
		}
		a.jetonIlerlet()
	}
	return olusturulanGorev
}

func (a *Ayristirici) komutuAyristir() *Komut {
	var k Komut
	if a.suAnkiJeton.Tipi == JetonExec || a.suAnkiJeton.Tipi == JetonPrint {
		k.KomutTuru = a.suAnkiJeton.Degeri
		a.jetonIlerlet()
		
		if a.suAnkiJeton.Tipi != JetonMetin {
			a.hataListesi = append(a.hataListesi, fmt.Sprintf("Satır %d: Komuttan sonra tırnak içinde metin yazılmalı.", a.suAnkiJeton.SatirNo))
			return nil
		}
		k.Arguman = a.suAnkiJeton.Degeri
		a.jetonIlerlet()
		
		if a.suAnkiJeton.Tipi != JetonNoktaliVirgül {
			a.hataListesi = append(a.hataListesi, fmt.Sprintf("Satır %d: Komutun sonuna noktalı virgül (;) koymayı unutmuşsunuz.", a.suAnkiJeton.SatirNo))
		}
		return &k
	}
	
	a.hataListesi = append(a.hataListesi, fmt.Sprintf("Satır %d: Geçersiz komut '%s'", a.suAnkiJeton.SatirNo, a.suAnkiJeton.Degeri))
	return nil
}

// --- Çalıştırıcı (Executor) Kısmı ---

type Calistirici struct {
	gorevSözlügü map[string]Gorev
	durumTablosu map[string]string // "bekliyor", "calisiyor", "bitti", "hata"
	durumKilidi  sync.Mutex // goroutine'ler aynı anda tabloyu bozmasın diye kilit koyuyorum
	bekleyici    sync.WaitGroup
	hataKanali   chan error
}

func CalistiriciOlustur(gorevler []Gorev) *Calistirici {
	c := &Calistirici{
		gorevSözlügü: make(map[string]Gorev),
		durumTablosu: make(map[string]string),
		hataKanali:   make(chan error, len(gorevler)),
	}
	
	for _, g := range gorevler {
		c.gorevSözlügü[g.GorevAdi] = g
		c.durumTablosu[g.GorevAdi] = "bekliyor"
	}
	return c
}

func (c *Calistirici) Baslat() error {
	// Her bir task için ayrı bir goroutine açıp paralel çalıştırıyorum
	for ad := range c.gorevSözlügü {
		c.bekleyici.Add(1)
		go c.arkaPlandaCalistir(ad)
	}
	
	c.bekleyici.Wait() // Herkesin bitmesini bekle
	close(c.hataKanali)

	var hatalar []string
	for h := range c.hataKanali {
		hatalar = append(hatalar, h.Error())
	}
	
	if len(hatalar) > 0 {
		return fmt.Errorf(strings.Join(hatalar, "\n"))
	}
	return nil
}

func (c *Calistirici) arkaPlandaCalistir(gorevAdi string) {
	defer c.bekleyici.Done() // Fonksiyon bitince waitgroup sayacını düşür
	
	mevcutGorev, varMi := c.gorevSözlügü[gorevAdi]
	if !varMi {
		c.hataKanali <- fmt.Errorf("Gorev bulunamadi: %s", gorevAdi)
		return
	}

	// 1) Bağımlılıkların bitmesini bekleyelim (Polling yapıyoruz)
	for {
		baslayabilirMiyim := true
		c.durumKilidi.Lock()
		
		for _, bagimlilik := range mevcutGorev.Bagimliliklar {
			bDurum := c.durumTablosu[bagimlilik]
			
			// Eğer beklediğim task patladıysa beni de iptal et
			if bDurum == "hata" {
				c.durumTablosu[gorevAdi] = "hata"
				c.durumKilidi.Unlock()
				c.hataKanali <- fmt.Errorf("Iptal: '%s' gorevi iptal edildi. Cunku bagli oldugu '%s' gorevi hata verdi.", gorevAdi, bagimlilik)
				return
			}
			
			if bDurum != "bitti" {
				baslayabilirMiyim = false
				break
			}
		}
		c.durumKilidi.Unlock()
		
		if baslayabilirMiyim {
			break // Oh be, sonunda çalışabiliriz
		}
		
		// CPU'yu yormamak için biraz uyuyalım
		time.Sleep(50 * time.Millisecond)
	}

	// 2) Çalışmaya başlıyoruz
	c.durumKilidi.Lock()
	c.durumTablosu[gorevAdi] = "calisiyor"
	c.durumKilidi.Unlock()

	fmt.Printf("[BASLADI] Görev '%s' çalışıyor...\n", gorevAdi)
	
	for _, kmt := range mevcutGorev.KomutListesi {
		if kmt.KomutTuru == "print" {
			fmt.Printf("--> [%s] %s\n", gorevAdi, kmt.Arguman)
		} else if kmt.KomutTuru == "exec" {
			// Komutu bosluklara göre parçalayıp işletim sistemine yolluyorum
			parcalar := strings.Fields(kmt.Arguman)
			if len(parcalar) > 0 {
				isletimSistemiKomutu := exec.Command(parcalar[0], parcalar[1:]...)
				isletimSistemiKomutu.Stdout = os.Stdout
				isletimSistemiKomutu.Stderr = os.Stderr
				
				hata := isletimSistemiKomutu.Run()
				if hata != nil {
					c.durumKilidi.Lock()
					c.durumTablosu[gorevAdi] = "hata"
					c.durumKilidi.Unlock()
					c.hataKanali <- fmt.Errorf("HATA: '%s' icindeki komut calismadi: %v", gorevAdi, hata)
					return
				}
			}
		}
	}

	// 3) Bitiş
	c.durumKilidi.Lock()
	c.durumTablosu[gorevAdi] = "bitti"
	c.durumKilidi.Unlock()
	fmt.Printf("[TAMAMLANDI] Görev '%s' bitti.\n", gorevAdi)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Hatalı kullanim! Soyle calistirin: taskrunner <dosya.tr>")
		return
	}
	
	okunanIcerik, hata := os.ReadFile(os.Args[1])
	if hata != nil {
		fmt.Println("Dosya okunamadi, hata:", hata)
		return
	}

	// Parser adımları
	sok := OkuyucuOlustur(string(okunanIcerik))
	ayr := AyristiriciOlustur(sok)
	tumGorevler := ayr.ProgramiAyristir()

	if len(ayr.hataListesi) > 0 {
		fmt.Println("Kodda yazim hatalari buldum:")
		for _, h := range ayr.hataListesi {
			fmt.Println("- ", h)
		}
		return
	}

	fmt.Println("---- Islemler basliyor ----")
	calistirici := CalistiriciOlustur(tumGorevler)
	hata = calistirici.Baslat()
	
	if hata != nil {
		fmt.Println("\nMaalesef calisma sirasinda bazi sorunlar cikti:")
		fmt.Println(hata)
	} else {
		fmt.Println("\nHer sey sorunsuz sekilde tamamlandi!")
	}
}
