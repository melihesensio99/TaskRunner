# TaskRunner Mini Programlama Dili Raporu


Bu ödevde, yazdığımız senaryolara göre taskleri sırasıyla (ya da paralel olarak) çalıştıracak ufak bir programlama dili ve onun yorumlayıcısını (interpreter) geliştirdim. Dili yazarken Go'yu tercih ettim çünkü Go'nun goroutine'leri sayesinde bu tarz asenkron işlemleri yönetmek nispeten daha az karmaşık oluyordu.

## 1. Tasarım Kararları ve Çözüm Yaklaşımım

Ödevin asıl zor kısmı kodu sadece satır satır okumak değil, hangi görevlerin birbirini beklemesi gerektiğini ayarlamaktı. Bunu çözmek için klasik compiler derslerindeki Lexer ve Parser mantığını basitleştirerek kendim yazdım. Ekstra bir kütüphane (Lex/Yacc gibi) kullanmadım ki bağımlılık olmasın.
- **Sözcük Okuyucu (Lexer):** Kodu karakter karakter okuyup "Jeton" (Token) dizisine çeviriyor.
- **Ayrıştırıcı (Parser):** Bu jetonları alıp "Görev" (Task) yapılarına atıyor.
- **Çalıştırıcı (Executor):** Tüm işi yapan asıl kısım burası. Her task için bir tane `goroutine` açıp paralel çalıştırıyorum. Eğer task'ın `depends` kısmında beklediği bir şey varsa, o bitene kadar for döngüsü içinde bekletiyorum (polling).

## 2. Karşılaştığım Zorluklar

- **Değişkenleri korumak:** Herkes (tüm goroutine'ler) aynı anda "benim işim bitti", "ben bekliyorum" diye durum değiştirmeye kalkınca program sapıtmaya başlıyordu (race condition). Bunu engellemek için kodun ilgili yerlerine `sync.Mutex` kilitleri koydum. Böylece durumu sadece bir goroutine güncelleyebiliyor.
- **Zincirleme hata iptalleri:** Bir görevin içi çökerse (örneğin işletim sistemi komutu hata verirse), onu bekleyen görevlerin sonsuza kadar beklememesi gerekiyordu. Bu yüzden hata kontrolü koydum. Bağımlı olunan task çöktüyse, diğer taskler de ekrana "iptal edildi" yazıp kendini kapatıyor.

## 3. Test Sonuçlarım

Ödev yönergesindeki gibi 3 farklı test dosyası hazırladım ve klasöre koydum:

1. **`test1_normal.tr` (Normal Durum):** En temel senaryom. İki tane alt task aynı anda başlıyor. Biri bitince ana task (compile) başlıyor vb. İstediğim gibi tüm akış sorunsuz tamamlandı.
2. **`test2_boundary.tr` (Sınır Durumu):** Sistemin bağımlılık olmadan tek bir komutla patlayıp patlamadığını görmek için oluşturduğum test. İçinde hiç `depends` kullanmadım, kod yine hata vermeden çalıştı.
3. **`test3_error.tr` (Hata Durumu):** Dilimin kurallarına göre komutların sonuna noktalı virgül (`;`) koymak zorunlu. Kasıtlı olarak bunu unuttuğum bir senaryo hazırladım. Parser bunu görünce ekrana "Satır 3: Komutun sonuna noktalı virgül (;) koymayı unutmuşsunuz" hatasını basıp çalışmayı güvenli şekilde kesti.
