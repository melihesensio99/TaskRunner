=================================================
Ders: Programlama Dilleri Prensipleri
Öğrenci: Melih Esen
Numara: 2210656013
Ödev No: 112
Tarih: 23.05.2026
=================================================

PROGRAMLAMA DİLİ VE SÜRÜM:
  Go 1.20+ (Herhangi bir standart Go sürümü çalıştırabilir)

GEREKLİ BAĞIMLILIKLAR:
  Sadece Go standart kütüphaneleri kullanılmıştır. Harici paket (external dependency) yoktur.

DERLEME KOMUTU (gerekiyorsa):
  go build -o taskrunner.exe src/main.go

ÇALIŞTIRMA KOMUTU:
  ./taskrunner.exe tests/test1_normal.tr
  (veya doğrudan çalıştırmak için: go run src/main.go tests/test1_normal.tr)

ÖRNEK KULLANIM:
  1. Terminali proje dizininde (README.txt'nin bulunduğu dizin) açın.
  2. "go run src/main.go tests/test1_normal.tr" komutunu yazarak normal test senaryosunu çalıştırın.
  3. Ekranda paralel görevlerin başlatılması, birbirini beklemesi ve sonuçlanması durumlarını gözlemleyin.
  
  Ayrıca diğer test senaryolarını da test edebilirsiniz:
  - Sınır Durumu (Boundary): go run src/main.go tests/test2_boundary.tr
  - Hata Durumu (Error): go run src/main.go tests/test3_error.tr

BİLİNEN SORUNLAR (varsa):
  - Bağımlılıklarda oluşabilecek potansiyel sonsuz döngüler (cyclic dependency) için bir kontrol mekanizması eklenmemiştir. Yazılan senaryolarda doğru hiyerarşi kurulmalıdır.
  - Hata yakalama işlemi (error handling) sadece hata durumunda ekrana mesaj basma ve bağımlı task'leri durdurma şeklinde basitleştirilmiştir.
