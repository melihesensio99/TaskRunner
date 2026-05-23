# TaskRunner Mini Programlama Dili Raporu

Bu rapor, masaüstünüzde oluşturulan `taskrunner` projesi (Go ile yazılmış mini bir task runner dili ve yorumlayıcısı) hakkında detayları içerir. Sisteminizde şu an Go kurulu olmadığı için kodlar derlenip çalıştırılamasa da, kaynak kodlar (`main.go`) ve örnek program (`example.tr`) masaüstünüzdeki `taskrunner` klasöründe hazır durumdadır.

## 1. BNF Grameri

Dilin tasarımı, birbiriyle bağımlılıkları olan görevleri (task) tanımlamaya odaklanmıştır.

```bnf
<program> ::= <task_decl>*
<task_decl> ::= "task" <identifier> [ "depends" <identifier_list> ] "{" <statement>* "}"
<identifier_list> ::= <identifier> ( "," <identifier> )*
<statement> ::= <command> <string_literal> ";"
<command> ::= "exec" | "print"
```

## 2. Dilin Özellikleri

- **Görev Tanımlama ve Bağımlılık (Task Definition & Dependencies):** Her `task` benzersiz bir isme sahiptir. İsteğe bağlı olarak `depends` anahtar kelimesi ile başka görevlere olan bağımlılıklar belirtilebilir. Virgülle ayrılarak birden fazla bağımlılık eklenebilir.
- **Paralel Çalıştırma (Parallel Task Execution):** Yorumlayıcı (`Executor`), tüm görevleri Go goroutine'leri aracılığıyla eşzamanlı olarak başlatır. Bağımlılığı olan görevler, bağımlı oldukları görevler başarıyla tamamlanana kadar beklerler. Bağımlılığı olmayan görevler (örneğin aşağıdaki örnekte `compile` ve `lint`) aynı anda çalışmaya başlar.
- **Temel Hata Raporlama:** Sözdizimi (Syntax) hataları ayrıştırma (parsing) aşamasında satır numarası ile birlikte raporlanır. Çalışma zamanı hataları veya herhangi bir `exec` komutunun hata kodu döndürmesi durumunda, bu durum bağımlı görevlere yansıtılır ve o görevler de iptal edilip raporlanır.

## 3. Örnek Program (`example.tr`)

Aşağıda masaüstünüzdeki dizinde de bulunan örnek betik yer almaktadır:

```text
task compile {
    print "Compiling source code...";
    exec "echo compile_success";
}

task test depends compile {
    print "Running unit tests...";
    exec "echo tests_passed";
}

task lint {
    print "Linting source files...";
    exec "echo lint_passed";
}

task deploy depends test, lint {
    print "Deploying application...";
    exec "echo deploy_success";
}
```

## 4. Beklenen Çalıştırma Sonucu (Execution Output)

Eğer sisteminizde Go kurulu olsaydı `go run main.go example.tr` komutunu çalıştırdığınızda aşağıdaki gibi bir asenkron çalışma çıktısı alacaktınız. (Çıktı sırası paralel çalıştığı için bilgisayarın durumuna göre değişebilir, ancak `test` daima `compile` bittikten sonra, `deploy` ise daima `test` ve `lint` bittikten sonra çalışacaktır.)

```text
Starting execution...
[RUNNING] Task 'lint' started
[lint] Linting source files...
[RUNNING] Task 'compile' started
[compile] Compiling source code...
lint_passed
[DONE] Task 'lint' finished
compile_success
[DONE] Task 'compile' finished
[RUNNING] Task 'test' started
[test] Running unit tests...
tests_passed
[DONE] Task 'test' finished
[RUNNING] Task 'deploy' started
[deploy] Deploying application...
deploy_success
[DONE] Task 'deploy' finished

All tasks completed successfully!
```

---

**Not:** Bu projeyi çalıştırmak için [go.dev](https://go.dev) adresinden Go programlama dilini kurabilir, daha sonra masaüstünüzdeki `taskrunner` klasörü içinde bir terminal açarak `go run main.go example.tr` yazabilirsiniz.
