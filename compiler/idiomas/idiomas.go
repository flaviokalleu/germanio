package idiomas

// Translations maps words from any language to the canonical Portuguese keyword.
// The lexer uses this to normalize all input to a common set of tokens.
// Each entry: "foreign_word" → "canonical_pt_keyword"
//
// To add a new language: just add entries to the map.
// The canonical keywords are the Portuguese ones used internally.

var Translations = map[string]string{

	// ===================== ESPAÑOL (ES) =====================
	// ~560 million speakers
	"datos": "dados", "pantallas": "telas", "pantalla": "tela",
	"acciones": "acoes", "integraciones": "integracoes",
	"base_de_datos": "banco", "autenticacion": "autenticacao",
	"importar_es": "_skip", // same as PT
	"titulo_es":   "_skip", // same as PT
	"mostrar_es":  "_skip", // same as PT
	"boton":       "botao", "formulario_es": "_skip",
	"entrada_es": "_skip", "busqueda": "busca",
	"grafico_es": "_skip", "tabla": "tabela",
	"campo_es": "_skip", "seleccionar": "selecionar",
	"area_de_texto": "area_texto",
	// Types ES
	"texto_es": "_skip", "numero_es": "_skip",
	"fecha": "data", "booleano_es": "_skip",
	"correo": "email", "telefono": "telefone",
	"imagen": "imagem", "archivo_es": "_skip",
	"enlace": "link", "estado": "status",
	"dinero": "dinheiro", "contrasena": "senha",
	"texto_largo": "texto_longo",
	// Events ES
	"cuando": "quando", "clic": "clicar", "hacer_clic": "clicar",
	"actualizar": "atualizar", "eliminar": "deletar",
	// Logic ES
	"si": "se", "sino": "senao", "entonces": "entao",
	"mayor": "maior", "menor_es": "_skip",
	"y": "e", "o": "ou", "no": "nao",
	"definir_es": "_skip", "devolver": "retornar", "cambiar": "mudar",
	"para_es": "_skip", "para_cada_es": "_skip",
	"funcion": "funcao", "intentar": "tentar",
	"obligatorio": "obrigatorio", "unico_es": "_skip",
	"predeterminado": "padrao",
	// Auth ES
	"registro_es": "_skip", "permiso": "permissao",
	"requiere": "requer", "publico_es": "_skip",
	// Scripting ES
	"mientras": "enquanto", "veces": "vezes",
	"nulo_es": "_skip", "verdadero": "verdadeiro",
	"falso_es": "_skip", "hasta": "ate",
	"romper": "parar", "continuar_es": "_skip",
	"detener": "parar", "imprimir": "mostrar",
	// Relationships ES
	"pertenece_a": "pertence_a", "tiene_muchos": "tem_muitos",
	"muchos_a_muchos": "muitos_para_muitos",
	// Theme ES
	"oscuro": "escuro", "claro_es": "_skip",
	"fuente": "fonte", "borde": "borda", "fondo": "fundo",
	"tarjeta": "cartao", "estilo_es": "_skip",
	"mensaje": "mensagem", "notificar_es": "_skip",
	"llamar": "chamar", "pago": "pagamento",
	"cada_es": "_skip", "hora_es": "_skip", "minuto_es": "_skip",
	"repetir_es": "_skip",

	// ===================== FRANÇAIS (FR) =====================
	// ~320 million speakers
	"systeme": "sistema", "donnees": "dados", "ecrans": "telas",
	"ecran": "tela", "evenements": "eventos", "evenement": "eventos",
	"actions_fr": "_skip", "integrations_fr": "_skip",
	"logique": "logica", "base_donnees": "banco",
	"authentification": "autenticacao",
	"importer":         "importar",
	"titre":            "titulo", "afficher": "mostrar",
	"bouton": "botao", "formulaire": "formulario",
	"recherche": "busca", "graphique": "grafico",
	"tableau": "tabela", "champ": "campo",
	// Types FR
	"texte": "texto", "nombre": "numero", "argent": "dinheiro",
	"mot_de_passe": "senha", "fichier": "arquivo",
	"lien": "link", "statut": "status", "booleen": "booleano",
	"texte_long": "texto_longo",
	// Events FR
	"quand": "quando", "cliquer": "clicar",
	"creer": "criar", "mettre_a_jour": "atualizar",
	"supprimer": "deletar", "envoyer": "enviar",
	// Logic FR
	"sinon": "senao", "egal": "igual",
	"plus_grand": "maior", "plus_petit": "menor",
	"et": "e", "non": "nao", "alors": "entao",
	"valider": "validar", "calculer": "calcular",
	"retourner": "retornar", "changer": "mudar",
	"pour_chaque": "para_cada", "fonction": "funcao",
	"essayer": "tentar", "erreur": "erro",
	// Auth FR
	"inscription": "registro", "utilisateur": "usuario",
	"autorisation": "permissao", "exiger": "requer",
	// Modifiers FR
	"obligatoire": "obrigatorio",
	"defaut":      "padrao",
	// Scripting FR
	"tant_que": "enquanto", "fois": "vezes",
	"vrai": "verdadeiro", "faux": "falso",
	"arreter": "parar", "continuer": "continuar",
	"afficher_fr": "_skip",
	// Relationships FR
	"appartient_a": "pertence_a", "a_plusieurs": "tem_muitos",
	"plusieurs_a_plusieurs": "muitos_para_muitos",
	// Theme FR
	"couleur": "cor", "sombre": "escuro", "clair": "claro",
	"police": "fonte", "bordure": "borda", "fond": "fundo",
	"carte": "cartao", "style_fr": "_skip",
	"icone_fr": "_skip",

	// ===================== DEUTSCH (DE) =====================
	// ~130 million speakers
	"daten": "dados", "bildschirme": "telas",
	"bildschirm": "tela", "ereignisse": "eventos",
	"aktionen": "acoes", "integrationen": "integracoes",
	"datenbank": "banco", "authentifizierung": "autenticacao",
	"importieren": "importar", "von": "de",
	"anzeigen": "mostrar", "knopf": "botao",
	"formular": "formulario", "suche": "busca",
	"diagramm": "grafico", "tabelle": "tabela", "feld": "campo",
	// Types DE
	"zahl": "numero", "geld": "dinheiro",
	"passwort": "senha", "datei": "arquivo",
	"bild": "imagem", "verknuepfung": "link",
	"langer_text": "texto_longo", "datum": "data",
	"wahrheitswert": "booleano", "telefon": "telefone",
	// Events DE
	"wann": "quando", "klicken": "clicar",
	"erstellen": "criar", "aktualisieren": "atualizar",
	"loeschen": "deletar", "senden": "enviar",
	// Logic DE
	"wenn": "se", "sonst": "senao", "gleich": "igual",
	"groesser": "maior", "kleiner": "menor",
	"und": "e", "oder": "ou", "nicht": "nao",
	"dann": "entao", "validieren": "validar",
	"berechnen": "calcular", "festlegen": "definir",
	"zurueckgeben": "retornar", "aendern": "mudar",
	"fuer_jedes": "para_cada", "funktion": "funcao",
	"versuchen": "tentar", "fehler": "erro",
	// Auth DE
	"registrierung": "registro", "benutzer": "usuario",
	"berechtigung": "permissao", "erfordert": "requer",
	"oeffentlich": "publico",
	// Modifiers DE
	"pflichtfeld": "obrigatorio", "eindeutig": "unico",
	"standard": "padrao",
	// Scripting DE
	"solange": "enquanto", "wiederholen": "repetir", "mal": "vezes",
	"wahr": "verdadeiro", "falsch": "falso",
	"anhalten": "parar", "weiter": "continuar",
	"drucken": "mostrar",
	// Relationships DE
	"gehoert_zu": "pertence_a", "hat_viele": "tem_muitos",
	"viele_zu_viele": "muitos_para_muitos",
	// Theme DE
	"farbe": "cor", "dunkel": "escuro", "hell": "claro",
	"schriftart": "fonte", "rand": "borda",
	"hintergrund": "fundo", "karte": "cartao",
	"stil": "estilo", "symbol": "icone",
	"nachricht": "mensagem", "benachrichtigen": "notificar",
	"aufrufen": "chamar", "zahlung": "pagamento",
	"jede": "cada", "stunde": "hora",

	// ===================== ITALIANO (IT) =====================
	// ~85 million speakers
	"dati": "dados", "schermate": "telas",
	"schermata": "tela", "eventi": "eventos",
	"azioni": "acoes", "integrazioni": "integracoes",
	"database_it":    "_skip",
	"autenticazione": "autenticacao",
	"importare":      "importar", "da": "de",
	"schermo": "tela", "elenco": "lista",
	"mostrare": "mostrar", "pulsante": "botao",
	"modulo": "formulario", "ricerca": "busca",
	"grafico_it": "_skip", "tabella": "tabela",
	// Types IT
	"testo": "texto", "intero": "numero",
	"soldi": "dinheiro", "parola_chiave": "senha",
	"archivio": "arquivo", "collegamento": "link",
	"testo_lungo": "texto_longo",
	// Events IT
	"quando_it": "_skip", "cliccare": "clicar",
	"aggiornare": "atualizar", "cancellare": "deletar",
	"inviare": "enviar",
	// Logic IT
	"allora": "entao", "altrimenti": "senao",
	"uguale": "igual", "maggiore": "maior", "minore": "menor",
	"validare": "validar", "calcolare": "calcular",
	"restituire": "retornar", "cambiare": "mudar",
	"per_ogni": "para_cada", "funzione": "funcao",
	"provare": "tentar", "errore": "erro",
	// Auth IT
	"registrazione": "registro", "utente": "usuario",
	"permesso": "permissao", "richiede": "requer",
	"pubblico": "publico",
	// Modifiers IT
	"obbligatorio": "obrigatorio",
	"predefinito":  "padrao",
	// Scripting IT
	"mentre": "enquanto", "volte": "vezes",
	"vero":     "verdadeiro",
	"fermare":  "parar",
	"stampare": "mostrar",
	// Relationships IT
	"appartiene_a": "pertence_a", "ha_molti": "tem_muitos",
	"molti_a_molti": "muitos_para_muitos",
	// Theme IT
	"colore": "cor", "scuro": "escuro", "chiaro": "claro",
	"carattere": "fonte", "bordo": "borda", "sfondo": "fundo",
	"scheda": "cartao",

	// ===================== 中文 CHINESE (ZH) =====================
	// ~1.1 billion speakers
	"系统": "sistema", "数据": "dados", "界面": "telas",
	"页面": "tela", "事件": "eventos", "操作": "acoes",
	"集成": "integracoes", "主题": "tema", "逻辑": "logica",
	"数据库": "banco", "认证": "autenticacao",
	"导入": "importar", "从": "de",
	"标题": "titulo", "列表": "lista", "显示": "mostrar",
	"按钮": "botao", "表单": "formulario", "搜索": "busca",
	"图表": "grafico", "表格": "tabela", "字段": "campo",
	// Types ZH
	"文本": "texto", "数字": "numero", "日期": "data",
	"布尔": "booleano", "邮箱": "email", "电话": "telefone",
	"图片": "imagem", "文件": "arquivo", "链接": "link",
	"状态": "status", "金额": "dinheiro", "密码": "senha",
	"长文本": "texto_longo", "枚举": "enum",
	// Events ZH
	"当": "quando", "点击": "clicar", "创建": "criar",
	"更新": "atualizar", "删除": "deletar", "发送": "enviar",
	// Logic ZH
	"如果": "se", "否则": "senao", "等于": "igual",
	"大于": "maior", "小于": "menor",
	"和": "e", "或": "ou", "不": "nao", "那么": "entao",
	"验证": "validar", "计算": "calcular",
	"定义": "definir", "返回": "retornar", "改变": "mudar",
	"遍历": "para_cada", "函数": "funcao",
	"尝试": "tentar", "错误": "erro",
	// Auth ZH
	"登录": "login", "注册": "registro", "用户": "usuario",
	"权限": "permissao", "需要": "requer", "管理员": "admin",
	"公开": "publico",
	// Modifiers ZH
	"必填": "obrigatorio", "唯一": "unico", "默认": "padrao",
	"索引": "indice",
	// Scripting ZH
	"暂停": "pausar", "继续": "continuar", "停止": "parar",
	"循环": "enquanto", "重复": "repetir", "次": "vezes",
	"空": "nulo", "真": "verdadeiro", "假": "falso",
	"打印": "mostrar",
	// Relationships ZH
	"属于": "pertence_a", "拥有多个": "tem_muitos",
	"多对多": "muitos_para_muitos",
	// Theme ZH
	"颜色": "cor", "深色": "escuro", "浅色": "claro",
	"字体": "fonte", "圆角": "borda", "背景": "fundo",
	"卡片": "cartao", "风格": "estilo", "图标": "icone",
	"消息": "mensagem", "通知": "notificar", "调用": "chamar",
	"支付": "pagamento", "每": "cada", "小时": "hora", "分钟": "minuto",

	// ===================== 日本語 JAPANESE (JA) =====================
	// ~125 million speakers
	"システム": "sistema", "データ": "dados", "画面一覧": "telas",
	"画面": "tela", "イベント": "eventos", "アクション": "acoes",
	"テーマ": "tema", "ロジック": "logica",
	"データベース": "banco", "認証": "autenticacao",
	"インポート": "importar",
	"タイトル":  "titulo", "リスト": "lista", "表示": "mostrar",
	"ボタン": "botao", "フォーム": "formulario", "検索": "busca",
	"グラフ": "grafico", "テーブル": "tabela",
	// Types JA
	"テキスト": "texto", "数値": "numero", "日付": "data",
	"真偽": "booleano", "メール": "email", "画像": "imagem",
	"ファイル": "arquivo", "リンク": "link",
	"ステータス": "status", "金額": "dinheiro", "パスワード": "senha",
	"長文": "texto_longo",
	// Events JA
	"クリック": "clicar", "作成": "criar",
	"更新する": "atualizar", "削除する": "deletar", "送信": "enviar",
	// Logic JA
	"もし": "se", "それ以外": "senao", "等しい": "igual",
	"より大きい": "maior", "より小さい": "menor",
	"かつ": "e", "または": "ou", "ではない": "nao",
	"定義する": "definir", "戻す": "retornar", "関数": "funcao",
	"試す": "tentar", "エラー": "erro",
	// Auth JA
	"ログイン": "login", "登録": "registro", "ユーザー": "usuario",
	"必須": "obrigatorio", "公開": "publico",
	// Scripting JA
	"繰り返す": "repetir", "回": "vezes", "停止する": "parar",
	"印刷": "mostrar",
	// Relationships JA
	"所属": "pertence_a", "複数所有": "tem_muitos",
	// Theme JA
	"色": "cor", "ダーク": "escuro", "ライト": "claro",
	"フォント": "fonte", "背景色": "fundo", "スタイル": "estilo",

	// ===================== 한국어 KOREAN (KO) =====================
	// ~80 million speakers
	"시스템": "sistema", "데이터": "dados", "화면들": "telas",
	"화면": "tela", "이벤트": "eventos", "테마": "tema",
	"로직": "logica", "데이터베이스": "banco", "인증": "autenticacao",
	"가져오기": "importar",
	"제목":   "titulo", "목록": "lista", "보기": "mostrar",
	"버튼": "botao", "양식": "formulario", "찾기": "busca",
	"차트": "grafico", "테이블": "tabela",
	// Types KO
	"텍스트": "texto", "숫자": "numero", "날짜": "data",
	"이메일": "email", "전화": "telefone", "이미지": "imagem",
	"파일": "arquivo", "상태": "status", "금액": "dinheiro",
	"비밀번호": "senha", "긴텍스트": "texto_longo",
	// Events KO
	"클릭": "clicar", "생성": "criar", "수정": "atualizar",
	"삭제": "deletar", "전송": "enviar",
	// Logic KO
	"만약": "se", "아니면": "senao", "같다": "igual",
	"크다": "maior", "작다": "menor",
	"그리고": "e", "또는": "ou", "아닌": "nao",
	"정의": "definir", "반환": "retornar", "함수": "funcao",
	// Auth KO
	"로그인": "login", "등록": "registro", "사용자": "usuario",
	"필수": "obrigatorio", "공개": "publico",
	// Scripting KO
	"반복": "repetir", "번": "vezes", "중지": "parar",
	"출력": "mostrar", "참": "verdadeiro", "거짓": "falso",

	// ===================== العربية ARABIC (AR) =====================
	// ~380 million speakers
	"نظام": "sistema", "بيانات": "dados", "شاشات": "telas",
	"شاشة": "tela", "احداث": "eventos", "سمة": "tema",
	"منطق": "logica", "قاعدة_بيانات": "banco", "مصادقة": "autenticacao",
	"استيراد": "importar", "من": "de",
	"عنوان": "titulo", "قائمة": "lista", "عرض": "mostrar",
	"زر": "botao", "نموذج": "formulario", "بحث": "busca",
	"رسم_بياني": "grafico", "جدول": "tabela",
	// Types AR
	"نص": "texto", "رقم": "numero", "تاريخ": "data",
	"بريد": "email", "هاتف": "telefone", "صورة": "imagem",
	"ملف": "arquivo", "رابط": "link", "حالة": "status",
	"مبلغ": "dinheiro", "كلمة_سر": "senha",
	"نص_طويل": "texto_longo",
	// Events AR
	"عند": "quando", "نقر": "clicar", "انشاء": "criar",
	"تحديث": "atualizar", "حذف": "deletar", "ارسال": "enviar",
	// Logic AR
	"اذا": "se", "والا": "senao", "يساوي": "igual",
	"اكبر": "maior", "اصغر": "menor",
	"و": "e", "او": "ou", "ليس": "nao",
	"تعريف": "definir", "ارجاع": "retornar", "دالة": "funcao",
	// Auth AR
	"تسجيل": "registro", "مستخدم": "usuario",
	"مطلوب": "obrigatorio", "عام": "publico",
	// Theme AR
	"لون": "cor", "داكن": "escuro", "فاتح": "claro",
	"خط": "fonte", "خلفية": "fundo",

	// ===================== HINDI (HI) =====================
	// ~600 million speakers
	"प्रणाली": "sistema", "डेटा": "dados", "स्क्रीन": "telas",
	"पृष्ठ": "tela", "घटनाएं": "eventos", "थीम": "tema",
	"तर्क": "logica", "डेटाबेस": "banco", "प्रमाणीकरण": "autenticacao",
	"आयात": "importar", "से": "de",
	"शीर्षक": "titulo", "सूची": "lista", "दिखाएं": "mostrar",
	"बटन": "botao", "फॉर्म": "formulario", "खोज": "busca",
	"चार्ट": "grafico", "तालिका": "tabela",
	// Types HI
	"पाठ": "texto", "संख्या": "numero", "तारीख": "data",
	"ईमेल": "email", "फोन": "telefone", "चित्र": "imagem",
	"फाइल": "arquivo", "स्थिति": "status", "राशि": "dinheiro",
	"पासवर्ड": "senha",
	// Events HI
	"जब": "quando", "क्लिक": "clicar", "बनाएं": "criar",
	"अपडेट": "atualizar", "हटाएं": "deletar", "भेजें": "enviar",
	// Logic HI
	"अगर": "se", "नहीं_तो": "senao", "बराबर": "igual",
	"बड़ा": "maior", "छोटा": "menor",
	"और": "e", "या": "ou", "नहीं": "nao",
	"परिभाषित": "definir", "लौटाएं": "retornar", "फंक्शन": "funcao",
	// Auth HI
	"लॉगिन": "login", "पंजीकरण": "registro", "उपयोगकर्ता": "usuario",
	"आवश्यक": "obrigatorio", "सार्वजनिक": "publico",

	// ===================== BENGALI (BN) =====================
	// ~270 million speakers
	"সিস্টেম": "sistema", "তথ্য": "dados", "পর্দা": "tela",
	"ইভেন্ট": "eventos", "থিম": "tema", "যুক্তি": "logica",
	"ডাটাবেস": "banco",
	"শিরোনাম": "titulo", "তালিকা": "lista", "দেখান": "mostrar",
	"বোতাম": "botao", "অনুসন্ধান": "busca",
	"লেখা": "texto", "সংখ্যা": "numero", "তারিখ": "data",
	"ছবি": "imagem", "অবস্থা": "status", "টাকা": "dinheiro",
	"যখন": "quando", "তৈরি": "criar",
	"যদি": "se", "নাহলে": "senao",
	"সত্য": "verdadeiro", "মিথ্যা": "falso",

	// ===================== РУССКИЙ RUSSIAN (RU) =====================
	// ~250 million speakers
	"система": "sistema", "данные": "dados", "экраны": "telas",
	"экран": "tela", "события": "eventos", "действия": "acoes",
	"интеграции": "integracoes", "тема": "tema", "логика": "logica",
	"база_данных": "banco", "аутентификация": "autenticacao",
	"импорт": "importar", "из": "de",
	"заголовок": "titulo", "список": "lista", "показать": "mostrar",
	"кнопка": "botao", "форма": "formulario", "поиск": "busca",
	"график": "grafico", "таблица": "tabela", "поле": "campo",
	// Types RU
	"текст": "texto", "число": "numero", "дата": "data",
	"логический": "booleano", "почта": "email",
	"изображение": "imagem", "файл": "arquivo",
	"ссылка": "link", "статус": "status", "деньги": "dinheiro",
	"пароль": "senha", "длинный_текст": "texto_longo",
	// Events RU
	"когда": "quando", "клик": "clicar", "создать": "criar",
	"обновить": "atualizar", "удалить": "deletar", "отправить": "enviar",
	// Logic RU
	"если": "se", "иначе": "senao", "равно": "igual",
	"больше": "maior", "меньше": "menor",
	"нет": "nao", "тогда": "entao",
	"определить": "definir", "вернуть": "retornar",
	"изменить": "mudar", "функция": "funcao",
	"попробовать": "tentar", "ошибка": "erro",
	// Auth RU
	"регистрация": "registro", "пользователь": "usuario",
	"разрешение": "permissao", "требует": "requer",
	"публичный": "publico",
	// Modifiers RU
	"обязательный": "obrigatorio", "уникальный": "unico",
	// Scripting RU
	"пока": "enquanto", "повторить": "repetir", "раз": "vezes",
	"истина": "verdadeiro", "ложь": "falso",
	"остановить": "parar", "продолжить": "continuar",
	"печать": "mostrar",
	// Relationships RU
	"принадлежит": "pertence_a", "имеет_много": "tem_muitos",
	"много_ко_многим": "muitos_para_muitos",
	// Theme RU
	"цвет": "cor", "темный": "escuro", "светлый": "claro",
	"шрифт": "fonte", "граница": "borda", "фон": "fundo",

	// ===================== PORTUGUÊS (PT) =====================
	// Already native — no translations needed

	// ===================== BAHASA INDONESIA (ID) =====================
	// ~270 million speakers
	"data_id": "_skip", "layar": "tela", "layar_layar": "telas",
	"peristiwa": "eventos", "tindakan": "acoes",
	"basis_data": "banco", "otentikasi": "autenticacao",
	"impor": "importar", "dari": "de",
	"judul": "titulo", "daftar": "lista", "tampilkan": "mostrar",
	"tombol": "botao", "cari": "busca",
	"teks": "texto", "angka": "numero", "tanggal": "data",
	"gambar": "imagem", "berkas": "arquivo",
	"uang": "dinheiro", "kata_sandi": "senha",
	"ketika": "quando", "klik": "clicar", "buat": "criar",
	"perbarui": "atualizar", "hapus": "deletar", "kirim": "enviar",
	"jika": "se", "lainnya": "senao",
	"sama": "igual", "lebih_besar": "maior", "lebih_kecil": "menor",
	"definisikan": "definir", "kembalikan": "retornar", "fungsi": "funcao",
	"wajib": "obrigatorio", "unik": "unico",
	"ulangi": "repetir", "kali": "vezes",
	"benar": "verdadeiro", "salah": "falso",
	"cetak": "mostrar", "hentikan": "parar",
	"milik": "pertence_a", "punya_banyak": "tem_muitos",
	"warna": "cor", "gelap": "escuro", "terang": "claro",

	// ===================== TÜRKÇE TURKISH (TR) =====================
	// ~85 million speakers
	"veri": "dados", "ekranlar": "telas", "ekran": "tela",
	"olaylar": "eventos", "eylemler": "acoes",
	"veritabani": "banco", "kimlik_dogrulama": "autenticacao",
	"iceri_aktar": "importar",
	"baslik":      "titulo", "goster": "mostrar",
	"dugme": "botao", "arama": "busca",
	"metin": "texto", "sayi": "numero", "tarih": "data",
	"resim": "imagem", "dosya": "arquivo",
	"durum": "status", "para": "dinheiro", "sifre": "senha",
	"olustur": "criar", "guncelle": "atualizar", "sil": "deletar",
	"gonder": "enviar",
	"eger":   "se", "degilse": "senao",
	"esit": "igual", "buyuk": "maior", "kucuk": "menor",
	"tanimla": "definir", "don": "retornar", "fonksiyon": "funcao",
	"zorunlu": "obrigatorio", "benzersiz": "unico",
	"tekrarla": "repetir", "kere": "vezes",
	"dogru": "verdadeiro", "yanlis": "falso",
	"yazdir": "mostrar", "durdur": "parar",
	"renk": "cor", "karanlik": "escuro", "aydin": "claro",

	// ===================== TIẾNG VIỆT VIETNAMESE (VI) =====================
	// ~85 million speakers
	"du_lieu": "dados", "man_hinh": "tela",
	"su_kien": "eventos", "chu_de": "tema",
	"co_so_du_lieu": "banco", "xac_thuc": "autenticacao",
	"nhap": "importar", "tu": "de",
	"tieu_de": "titulo", "danh_sach": "lista", "hien_thi": "mostrar",
	"nut": "botao", "tim_kiem": "busca",
	"van_ban": "texto", "so": "numero", "ngay": "data",
	"anh": "imagem", "tep": "arquivo",
	"trang_thai": "status", "tien": "dinheiro", "mat_khau": "senha",
	"tao": "criar", "cap_nhat": "atualizar", "xoa": "deletar",
	"gui": "enviar",
	"neu": "se", "nguoc_lai": "senao",
	"bang": "igual", "lon_hon": "maior", "nho_hon": "menor",
	"dinh_nghia": "definir", "tra_ve": "retornar", "ham": "funcao",
	"bat_buoc": "obrigatorio",
	"lap":      "repetir", "lan": "vezes",
	"dung": "verdadeiro", "sai": "falso",
	"in": "mostrar", "dung_lai": "parar",
	"mau": "cor", "toi": "escuro", "sang": "claro",

	// ===================== POLISH (PL) =====================
	// ~45 million speakers
	"dane": "dados", "ekrany": "telas",
	"zdarzenia": "eventos", "motyw": "tema",
	"baza_danych": "banco", "uwierzytelnianie": "autenticacao",
	"importuj": "importar", "z": "de",
	"tytul": "titulo", "pokaz": "mostrar",
	"przycisk": "botao", "szukaj": "busca",
	"tekst_pl": "_skip", "numer": "numero",
	"obraz": "imagem", "plik": "arquivo",
	"kwota": "dinheiro", "haslo": "senha",
	"utworz": "criar", "usun": "deletar", "wyslij": "enviar",
	"jezeli": "se", "inaczej": "senao",
	"rowne": "igual", "wieksze": "maior", "mniejsze": "menor",
	"zdefiniuj": "definir", "zwroc": "retornar", "funkcja": "funcao",
	"wymagane": "obrigatorio",
	"powtorz":  "repetir", "razy": "vezes",
	"prawda": "verdadeiro", "falsz": "falso",
	"wypisz": "mostrar", "zatrzymaj": "parar",
	"kolor": "cor", "ciemny": "escuro", "jasny": "claro",

	// ===================== DUTCH (NL) =====================
	// ~30 million speakers
	"gegevens": "dados", "schermen": "telas", "scherm": "tela",
	"gebeurtenissen": "eventos",
	"invoeren":       "importar", "van": "de",
	"tonen": "mostrar", "knop": "botao", "zoeken": "busca",
	"tekst_nl": "_skip", "nummer": "numero",
	"afbeelding": "imagem", "bestand": "arquivo",
	"bedrag": "dinheiro", "wachtwoord": "senha",
	"aanmaken": "criar", "bijwerken": "atualizar",
	"verwijderen": "deletar", "verzenden": "enviar",
	"als": "se", "anders": "senao",
	"gelijk": "igual", "groter": "maior",
	"definieer": "definir", "retourneer": "retornar", "functie": "funcao",
	"verplicht": "obrigatorio",
	"herhaal":   "repetir", "keer": "vezes",
	"waar": "verdadeiro", "onwaar": "falso",
	"afdrukken": "mostrar",
	"kleur":     "cor", "donker": "escuro", "licht": "claro",

	// ===================== THAI (TH) =====================
	// ~70 million speakers
	"ระบบ": "sistema", "ข้อมูล": "dados", "หน้าจอ": "tela",
	"เหตุการณ์": "eventos", "ธีม": "tema",
	"ฐานข้อมูล": "banco",
	"นำเข้า":    "importar", "จาก": "de",
	"หัวข้อ": "titulo", "รายการ": "lista", "แสดง": "mostrar",
	"ปุ่ม": "botao", "ค้นหา": "busca",
	"ข้อความ": "texto", "ตัวเลข": "numero", "วันที่": "data",
	"รูปภาพ": "imagem", "ไฟล์": "arquivo",
	"สถานะ": "status", "จำนวนเงิน": "dinheiro", "รหัสผ่าน": "senha",
	"สร้าง": "criar", "อัปเดต": "atualizar", "ลบ": "deletar",
	"ถ้า": "se", "ไม่เช่นนั้น": "senao",
	"เท่ากับ": "igual", "มากกว่า": "maior", "น้อยกว่า": "menor",
	"กำหนด": "definir", "คืนค่า": "retornar", "ฟังก์ชัน": "funcao",
	"จำเป็น": "obrigatorio",
	"พิมพ์":  "mostrar", "หยุด": "parar",
	"สี": "cor", "มืด": "escuro", "สว่าง": "claro",

	// ===================== SWAHILI (SW) =====================
	// ~100 million speakers (East Africa)
	"mfumo": "sistema", "takwimu": "dados", "skrini": "tela",
	"matukio": "eventos", "mandhari": "tema",
	"hifadhidata": "banco",
	"agiza":       "importar", "kutoka": "de",
	"kichwa": "titulo", "orodha": "lista", "onesha": "mostrar",
	"kitufe": "botao", "tafuta": "busca",
	"maandishi": "texto", "nambari": "numero", "tarehe": "data",
	"picha": "imagem", "faili": "arquivo",
	"hali": "status", "pesa": "dinheiro", "nenosiri": "senha",
	"tengeneza": "criar", "futa": "deletar", "tuma": "enviar",
	"ikiwa": "se", "vinginevyo": "senao",
	"sawa": "igual", "kubwa": "maior", "ndogo": "menor",
	"eleza": "definir", "rudisha": "retornar", "kazi": "funcao",
	"lazima":   "obrigatorio",
	"chapisha": "mostrar", "simama": "parar",
	"rangi": "cor", "giza": "escuro",
}
