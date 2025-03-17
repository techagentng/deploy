package db

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GormDB struct {
	DB *gorm.DB
}

func GetDB(c *config.Config) *GormDB {
	gormDB := &GormDB{}
	gormDB.Init(c)
	return gormDB
}

func (g *GormDB) Init(c *config.Config) {
	g.DB = getPostgresDB(c)

	if err := migrate(g.DB); err != nil {
		log.Fatalf("unable to run migrations: %v", err)
	}
}

func getPostgresDB(c *config.Config) *gorm.DB {
	log.Printf("Connecting to postgres: %+v", c)
	postgresDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d TimeZone=Africa/Lagos",
		c.PostgresHost, c.PostgresUser, c.PostgresPassword, c.PostgresDB, c.PostgresPort)

	// Create GORM DB instance
	gormConfig := &gorm.Config{}
	if c.Env != "prod" {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DSN: postgresDSN,
	}), gormConfig)
	if err != nil {
		log.Fatal(err)
	}

	return gormDB
}

func SeedRoles(db *gorm.DB) error {
	roles := []models.Role{
		{ID: uuid.New(), Name: "Admin"},
		{ID: uuid.New(), Name: "User"},
	}

	for _, role := range roles {
		if err := db.FirstOrCreate(&role, models.Role{Name: role.Name}).Error; err != nil {
			return err
		}
	}

	return nil
}

func toStringPtr(s string) *string {
    return &s
}

func SeedStates(db *gorm.DB) error {
	states := []models.State{
		{
			ID:            uuid.New(),
			State:         toStringPtr("Abia"),
			Governor:      toStringPtr("Alex Otti"),
			LGAC:          toStringPtr("Umuahia"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1XOX3lmQnbBMU4qIP6bDXBe_BvMQUim52/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1TO91cw1SskIwD2m_yCGSFVNnSWuj_s2d/view?usp=sharing"),
			Lgas:          []string{"Aba North", "Aba South", "Arochukwu", "Bende", "Ikwuano", "Isiala Ngwa North", "Isiala Ngwa South", "Isuikwuato", "Obi Ngwa", "Ohafia", "Osisioma Ngwa", "Ugwunagbo", "Ukwa East", "Ukwa West", "Umuahia North", "Umuahia South", "Umu Nneochi"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Adamawa"),
			Governor:      toStringPtr("Ahmadu Fintiri"),
			LGAC:          toStringPtr("Yola"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1VqkgX_84hJR8TnREs2zYRACrHKZ4l5mk/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1X3ugnSHm-YNu43H2KVJKN9S7YypBJh87/view?usp=sharing"),
			Lgas:          []string{"Demsa", "Fufore", "Ganye", "Girei", "Gombi", "Guyuk", "Hong", "Jada", "Lamurde", "Madagali", "Maiha", "Mayo-Belwa", "Michika", "Mubi North", "Mubi South", "Numan", "Shelleng", "Song", "Toungo", "Yola North", "Yola South"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Akwa Ibom"),
			Governor:      toStringPtr("Umo Eno"),
			LGAC:          toStringPtr("Uyo"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1-pnZkCYuPg_sSMWd3SHd5XncnVlDKYxY/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1fCWlNOKLU_uNLpMV8HjZPOVHMRhZ07j-/view?usp=sharing"),
			Lgas:          []string{"Abak", "Eastern Obolo", "Eket", "Esit Eket", "Essien Udim", "Etim Ekpo", "Etinan", "Ibeno", "Ibesikpo Asutan", "Ibiono Ibom", "Ika", "Ikono", "Ikot Abasi", "Ikot Ekpene", "Ini", "Itu", "Mbo", "Mkpat Enin", "Nsit Atai", "Nsit Ibom", "Nsit Ubium", "Obot Akara", "Okobo", "Onna", "Oron", "Oruk Anam", "Udung Uko", "Ukanafun", "Uruan", "Urue-Offong/Oruko", "Uyo"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Anambra"),
			Governor:      toStringPtr("Chukwuma Soludo"),
			LGAC:          toStringPtr("Awka"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1-R7SbgT34m0KIqwKf-C-zaPK-4bmOXN8/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1pS6WFSqhxwBNpvrJ44W9ijIVe6xnU1eo/view?usp=sharing"),
			Lgas:          []string{"Aguata", "Anambra East", "Anambra West", "Anaocha", "Awka North", "Awka South", "Ayamelum", "Dunukofia", "Ekwusigo", "Idemili North", "Idemili South", "Ihiala", "Njikoka", "Nnewi North", "Nnewi South", "Ogbaru", "Onitsha North", "Onitsha South", "Orumba North", "Orumba South", "Oyi"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Bauchi"),
			Governor:      toStringPtr("Bala Mohammed"),
			LGAC:          toStringPtr("Bauchi"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1KdQTri2UqqKRqXkOhpYLa2_njrP9FDNH/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1cgrMu1NX0eLDe6oEZt4w1f3v1ZTbyc47/view?usp=sharing"),
			Lgas:          []string{"Alkaleri", "Bauchi", "Bogoro", "Damban", "Darazo", "Dass", "Gamawa", "Ganjuwa", "Giade", "Itas/Gadau", "Jama’are", "Katagum", "Kirfi", "Misau", "Ningi", "Shira", "Tafawa Balewa", "Toro", "Warji", "Zaki"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Bayelsa"),
			Governor:      toStringPtr("Douye Diri"),
			LGAC:          toStringPtr("Yenagoa"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/18g7LNMIP7Eoqw9fro1zMs_bKFCcTyUHj/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1naYZOXZMm1XakfD9ckXskYLMilfjws39/view?usp=sharing"),
			Lgas:          []string{"Brass", "Ekeremor", "Kolokuma/Opokuma", "Nembe", "Ogbia", "Sagbama", "Southern Ijaw", "Yenagoa"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Benue"),
			Governor:      toStringPtr("Hyacinth Alia"),
			LGAC:          toStringPtr("Makurdi"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/13mx7-uJU2-vZF00ALTzBvRJj5ECioJPa/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1nZQgOnSHIsT9kpqGc2_kdlO6xG5OtCWC/view?usp=sharing"),
			Lgas:          []string{"Ado", "Agatu", "Apa", "Buruku", "Gboko", "Guma", "Gwer East", "Gwer West", "Katsina-Ala", "Konshisha", "Kwande", "Logo", "Makurdi", "Obi", "Ogbadibo", "Ohimini", "Oju", "Okpokwu", "Otukpo", "Tarka", "Ukum", "Ushongo", "Vandeikya"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Borno"),
			Governor:      toStringPtr("Babagana Zulum"),
			LGAC:          toStringPtr("Maiduguri"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1VqN9pjaPqIrW8RQjDkRO-7ABG9OL4yzK/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1AYEX0q7B9VcAdDIIrV69LwCQQkO1n2vM/view?usp=sharing"),
			Lgas:          []string{"Abadam", "Askira/Uba", "Bama", "Bayo", "Biu", "Chibok", "Damboa", "Dikwa", "Gubio", "Guzamala", "Gwoza", "Hawul", "Jere", "Kaga", "Kala/Balge", "Konduga", "Kukawa", "Kwaya Kusar", "Mafa", "Magumeri", "Maiduguri", "Marte", "Mobbar", "Monguno", "Ngala", "Nganzai", "Shani"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Cross River"),
			Governor:      toStringPtr("Bassey Otu"),
			LGAC:          toStringPtr("Calabar"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/11DUOgsa6CUgf6SWUPSvkcshVpODw3emh/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1gnZ0D5QV07YHvTDxHNlRSXm9wVgM5Oya/view?usp=sharing"),
			Lgas:          []string{"Abi", "Akamkpa", "Akpabuyo", "Bakassi", "Bekwarra", "Biase", "Boki", "Calabar Municipal", "Calabar South", "Etung", "Ikom", "Obanliku", "Obubra", "Obudu", "Odukpani", "Ogoja", "Yakurr", "Yala"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Delta"),
			Governor:      toStringPtr("Sheriff Oborevwori"),
			LGAC:          toStringPtr("Asaba"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1i8dkwiaQRwjOGmd138PaJp_VNkFHVznj/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/19q1d_AtQOjI1-4_jLESTWcyr_Rhj7GIP/view?usp=sharing"),
			Lgas:          []string{"Aniocha North", "Aniocha South", "Bomadi", "Burutu", "Ethiope East", "Ethiope West", "Ika North East", "Ika South", "Isoko North", "Isoko South", "Ndokwa East", "Ndokwa West", "Okpe", "Oshimili North", "Oshimili South", "Patani", "Sapele", "Udu", "Ughelli North", "Ughelli South", "Ukwuani", "Uvwie", "Warri North", "Warri South", "Warri South West"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Ebonyi"),
			Governor:      toStringPtr("Francis Nwifuru"),
			LGAC:          toStringPtr("Abakaliki"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1GI123JAXB0lm0mYmz398lwJ7EknOAbAw/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1CP-XtNNy9ZTcvRv9mv-nALj7-umaBCDU/view?usp=sharing"),
			Lgas:          []string{"Abakaliki", "Afikpo North", "Afikpo South", "Ebonyi", "Ezza North", "Ezza South", "Ikwo", "Ishielu", "Ivo", "Izzi", "Ohaozara", "Ohaukwu", "Onicha"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Edo"),
			Governor:      toStringPtr("Godwin Obaseki"),
			LGAC:          toStringPtr("Benin City"),
			GovernorImage: toStringPtr("https://example.com/obaseki.jpg"),
			LgacImage:     toStringPtr("https://example.com/benin.jpg"),
			Lgas:          []string{"Akoko-Edo", "Egor", "Esan Central", "Esan South East", "Esan West", "Etsako Central", "Etsako East", "Etsako West", "Igueben", "Ikpoba-Okha", "Oredo", "Orhionmwon", "Ovia North East", "Ovia South West", "Owan East", "Owan West", "Uhunmwonde"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Ekiti"),
			Governor:      toStringPtr("Biodun Oyebanji"),
			LGAC:          toStringPtr("Ado-Ekiti"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1vZWfq-cQJTExJCzgLmQut72KCfHqNFfX/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1FFwdO7iBkAhHerV2rlmwmmZQU4TcFtIv/view?usp=sharing"),
			Lgas:          []string{"Ado-Ekiti", "Efon", "Ekiti East", "Ekiti South West", "Ekiti West", "Emure", "Gbonyin", "Ido/Osi", "Ijero", "Ikere", "Ikole", "Ilejemeje", "Irepodun/Ifelodun", "Ise/Orun", "Moba", "Oye"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Enugu"),
			Governor:      toStringPtr("Peter Mbah"),
			LGAC:          toStringPtr("Enugu"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1GSg8XEMyC79Lb4XMNJEaM-qZxn5LZGxm/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1G19Cy1gsrKKbQ_V7UQnl8HqPgjmBljjP/view?usp=sharing"),
			Lgas:          []string{"Aninri", "Awgu", "Enugu East", "Enugu North", "Enugu South", "Ezeagu", "Igbo Etiti", "Igbo Eze North", "Igbo Eze South", "Isi Uzo", "Nkanu East", "Nkanu West", "Nsukka", "Oji River", "Udenu", "Udi", "Uzo-Uwani"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Gombe"),
			Governor:      toStringPtr("Inuwa Yahaya"),
			LGAC:          toStringPtr("Gombe"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/10HRkiN-DNxPb59Ey6CHmokDddCbgWOJC/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1mjKrGVs0N8VoigAZ8ARHI9EQiVHuGRVC/view?usp=sharing"),
			Lgas:          []string{"Akko", "Balanga", "Billiri", "Dukku", "Funakaye", "Gombe", "Kaltungo", "Kwami", "Nafada", "Shongom", "Yamaltu/Deba"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Imo"),
			Governor:      toStringPtr("Hope Uzodinma"),
			LGAC:          toStringPtr("Owerri"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/14QbuX8Mxz7L4iVNW6ey3Q8piiqXciWmT/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1NX2KmsCM1sqwbE77paCQu4xf8GpnZW6T/view?usp=sharing"),
			Lgas:          []string{"Aboh Mbaise", "Ahiazu Mbaise", "Ehime Mbano", "Ezinihitte Mbaise", "Ideato North", "Ideato South", "Ihitte/Uboma", "Ikeduru", "Isiala Mbano", "Isu", "Mbaitoli", "Ngor Okpala", "Njaba", "Nkwerre", "Nwangele", "Obowo", "Oguta", "Ohaji/Egbema", "Okigwe", "Orlu", "Orsu", "Oru East", "Oru West", "Owerri Municipal", "Owerri North", "Owerri West"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Jigawa"),
			Governor:      toStringPtr("Umar Namadi"),
			LGAC:          toStringPtr("Dutse"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1OjcK258GFw3-SQ95V3wOo27PLUYUH-_m/view?usp=sharing"),
			LgacImage:     toStringPtr("https://example.com/dutse.jpg"),
			Lgas:          []string{"Auyo", "Babura", "Birniwa", "Birnin Kudu", "Buji", "Dutse", "Gagarawa", "Garki", "Gumel", "Guri", "Gwaram", "Gwiwa", "Hadejia", "Jahun", "Kafin Hausa", "Kaugama", "Kazaure", "Kiri Kasama", "Kiyawa", "Maigatari", "Malam Madori", "Miga", "Ringim", "Roni", "Sule Tankarkar", "Taura", "Yankwashi"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Kaduna"),
			Governor:      toStringPtr("Uba Sani"),
			LGAC:          toStringPtr("Kaduna North"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1QIoQA7nhzn8eaB2hff_sIP6_ItyHMS0o/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1CJ98lHl9tywa6Se6ugBFiQeNlyP07FA-/view?usp=sharing"),
			Lgas:          []string{"Birnin Gwari", "Chikun", "Giwa", "Igabi", "Ikara", "Jaba", "Jema’a", "Kachia", "Kaduna North", "Kaduna South", "Kagarko", "Kajuru", "Kaura", "Kauru", "Kubau", "Kudan", "Lere", "Makarfi", "Sabon Gari", "Sanga", "Soba", "Zangon Kataf", "Zaria"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Kano"),
			Governor:      toStringPtr("Abba Kabir Yusuf"),
			LGAC:          toStringPtr("Kano Municipal"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1JPrQT5de8GwWV1U8OVI73Ugo449ekE5S/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1vn8TYw_KQIFx5J8joDZ9NhzUMEjSKn0f/view?usp=sharing"),
			Lgas:          []string{"Ajingi", "Albasu", "Bagwai", "Bebeji", "Bichi", "Bunkure", "Dala", "Dambatta", "Dawakin Kudu", "Dawakin Tofa", "Doguwa", "Fagge", "Gabasawa", "Garko", "Garun Mallam", "Gaya", "Gezawa", "Gwale", "Gwarzo", "Kabo", "Kano Municipal", "Karaye", "Kibiya", "Kiru", "Kumbotso", "Kunchi", "Kura", "Madobi", "Makoda", "Minjibir", "Nasarawa", "Rano", "Rimin Gado", "Rogo", "Shanono", "Sumaila", "Takai", "Tarauni", "Tofa", "Tsanyawa", "Tudun Wada", "Ungogo", "Warawa", "Wudil"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Katsina"),
			Governor:      toStringPtr("Dikko Radda"),
			LGAC:          toStringPtr("Katsina"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1OsAwAHJnbpVdiBcgsZm2F3VGu30t5lbm/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1jt5-nzIahFEX0nTmuZYInlQeTdCLRGNR/view?usp=sharing"),
			Lgas:          []string{"Bakori", "Batagarawa", "Batsari", "Baure", "Bindawa", "Charanchi", "Dandume", "Danja", "Dan Musa", "Daura", "Dutsi", "Dutsin Ma", "Faskari", "Funtua", "Ingawa", "Jibia", "Kafur", "Kaita", "Kankara", "Kankia", "Katsina", "Kurfi", "Kusada", "Mai’Adua", "Malumfashi", "Mani", "Mashi", "Matazu", "Musawa", "Rimi", "Sabuwa", "Safana", "Sandamu", "Zango"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Kebbi"),
			Governor:      toStringPtr("Nasir Idris"),
			LGAC:          toStringPtr("Birnin Kebbi"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1gSUWtk1q2ZpksApjbKOVi1rHNQtJYOB3/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1mRmi7uqwxnbjyJA4rqCxGRK2C4GoAiQA/view?usp=sharing"),
			Lgas:          []string{"Aleiro", "Arewa Dandi", "Argungu", "Augie", "Bagudo", "Birnin Kebbi", "Bunza", "Dandi", "Fakai", "Gwandu", "Jega", "Kalgo", "Koko/Besse", "Maiyama", "Ngaski", "Sakaba", "Shanga", "Suru", "Wasagu/Danko", "Yauri", "Zuru"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Kogi"),
			Governor:      toStringPtr("Ahmed Usman Ododo"),
			LGAC:          toStringPtr("Lokoja"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1RboKD-zpu3IUDqFPcNEEl8q4pv-3QIGM/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1WdDmch2S4s9_Hus6ako6y9nQU4WrRi4g/view?usp=sharing"),
			Lgas:          []string{"Adavi", "Ajaokuta", "Ankpa", "Bassa", "Dekina", "Ibaji", "Idah", "Igalamela-Odolu", "Ijumu", "Kabba/Bunu", "Kogi", "Lokoja", "Mopa-Muro", "Ofu", "Ogori/Magongo", "Okehi", "Okene", "Olamaboro", "Omala", "Yagba East", "Yagba West"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Kwara"),
			Governor:      toStringPtr("AbdulRahman AbdulRazaq"),
			LGAC:          toStringPtr("Ilorin"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1X5HD18ij5q_EjOjJGX-s2tOXKoRw2p54/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1hgsZxUXBkeyhpDoHJfsE5cb12UvICkQa/view?usp=sharing"),
			Lgas:          []string{"Asa", "Baruten", "Edu", "Ekiti", "Ifelodun", "Ilorin East", "Ilorin South", "Ilorin West", "Irepodun", "Isin", "Kaiama", "Moro", "Offa", "Oke Ero", "Oyun", "Pategi"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Lagos"),
			Governor:      toStringPtr("Babajide Sanwo-Olu"),
			LGAC:          toStringPtr("Ikeja"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/15bz1tOUIZ9JjRiJ8ivWpgoZvA1fMF_tC/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/19EDqjGe8Mk8oyZ10VN5nU1h5LsxBVHtB/view?usp=sharing"),
			Lgas:          []string{"Agege", "Ajeromi-Ifelodun", "Alimosho", "Amuwo-Odofin", "Apapa", "Badagry", "Epe", "Eti-Osa", "Ibeju-Lekki", "Ifako-Ijaiye", "Ikeja", "Ikorodu", "Kosofe", "Lagos Island", "Lagos Mainland", "Mushin", "Ojo", "Oshodi-Isolo", "Shomolu", "Surulere"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Nasarawa"),
			Governor:      toStringPtr("Abdullahi Sule"),
			LGAC:          toStringPtr("Lafia"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1cZRCspOB1z9SyyTMd-R9NQxS-__i4DUg/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1CML8fyhrT8PPzRN21rBZFUJEPqMMVQ3m/view?usp=sharing"),
			Lgas:          []string{"Akwanga", "Awe", "Doma", "Karu", "Keana", "Keffi", "Kokona", "Lafia", "Nasarawa", "Nasarawa Egon", "Obi", "Toto", "Wamba"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Niger"),
			Governor:      toStringPtr("Mohammed Bago"),
			LGAC:          toStringPtr("Minna"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1LxE79u85HCT9Wv8BZ-u38OVo4w-gyNTV/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1mYy1uH7Xk1Rdw7wYhHaVMo--nfcRyaCX/view?usp=sharing"),
			Lgas:          []string{"Agaie", "Agwara", "Bida", "Borgu", "Bosso", "Chanchaga", "Edati", "Gbako", "Gurara", "Katcha", "Kontagora", "Lapai", "Lavun", "Magama", "Mariga", "Mashegu", "Mokwa", "Munya", "Paikoro", "Rafi", "Rijau", "Shiroro", "Suleja", "Tafa", "Wushishi"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Ogun"),
			Governor:      toStringPtr("Dapo Abiodun"),
			LGAC:          toStringPtr("Abeokuta South"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/183IjH5P6vhJEsOk6_hnQnMUVYf-CWcBW/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1Rs4HDH2ALL4isMtjywZRO2tKDqUk5ISS/view?usp=sharing"),
			Lgas:          []string{"Abeokuta North", "Abeokuta South", "Ado-Odo/Ota", "Egbado North", "Egbado South", "Ewekoro", "Ifo", "Ijebu East", "Ijebu North", "Ijebu North East", "Ijebu Ode", "Ikenne", "Imeko Afon", "Ipokia", "Obafemi Owode", "Odeda", "Odogbolu", "Ogun Waterside", "Remo North", "Shagamu"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Ondo"),
			Governor:      toStringPtr("Lucky Aiyedatiwa"),
			LGAC:          toStringPtr("Akure"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1WH6096Ypu1664Tv6OXIElhnPaW8W6R4t/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1lpGyI_PtCA3GmJxEcrPqq1ezO9Bvjjs8/view?usp=sharing"),
			Lgas:          []string{"Akoko North East", "Akoko North West", "Akoko South East", "Akoko South West", "Akure North", "Akure South", "Ese Odo", "Idanre", "Ifedore", "Ilaje", "Ile Oluji/Okeigbo", "Irele", "Odigbo", "Okitipupa", "Ondo East", "Ondo West", "Ose", "Owo"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Osun"),
			Governor:      toStringPtr("Ademola Adeleke"),
			LGAC:          toStringPtr("Osogbo"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1WH6096Ypu1664Tv6OXIElhnPaW8W6R4t/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1_v_XE-IIuQ8fZX3tYWxy517IvXaUdd-o/view?usp=sharing"),
			Lgas:          []string{"Aiyedaade", "Aiyedire", "Atakumosa East", "Atakumosa West", "Boluwaduro", "Boripe", "Ede North", "Ede South", "Egbedore", "Ejigbo", "Ife Central", "Ife East", "Ife North", "Ife South", "Ifedayo", "Ifelodun", "Ila", "Ilesa East", "Ilesa West", "Irepodun", "Irewole", "Isokan", "Iwo", "Obokun", "Odo Otin", "Ola Oluwa", "Olorunda", "Oriade", "Orolu", "Osogbo"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Oyo"),
			Governor:      toStringPtr("Seyi Makinde"),
			LGAC:          toStringPtr("Ibadan North"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1bmlygAeuKSkq9aKBoQDfm3e1VSpv9QyL/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1QoPlQW88KIzRyoMVBIi4SV4WQgsjAHPv/view?usp=sharing"),
			Lgas:          []string{"Afijio", "Akinyele", "Atiba", "Atisbo", "Egbeda", "Ibadan North", "Ibadan North East", "Ibadan North West", "Ibadan South East", "Ibadan South West", "Ibarapa Central", "Ibarapa East", "Ibarapa North", "Ido", "Irepo", "Iseyin", "Itesiwaju", "Iwajowa", "Kajola", "Lagelu", "Ogbomosho North", "Ogbomosho South", "Ogo Oluwa", "Olorunsogo", "Oluyole", "Ona Ara", "Orelope", "Ori Ire", "Oyo East", "Oyo West", "Saki East", "Saki West", "Surulere"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Plateau"),
			Governor:      toStringPtr("Caleb Mutfwang"),
			LGAC:          toStringPtr("Jos"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1Xn819RjpzyBT8DmfRUz0POp3NDKy5LPn/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1CK4hjqUdnmuI4dr6tDBQotrjYv_JOgTl/view?usp=sharing"),
			Lgas:          []string{"Barkin Ladi", "Bassa", "Bokkos", "Jos East", "Jos North", "Jos South", "Kanam", "Kanke", "Langtang North", "Langtang South", "Mangu", "Mikang", "Pankshin", "Qua’an Pan", "Riyom", "Shendam", "Wase"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Rivers"),
			Governor:      toStringPtr("Siminalayi Fubara"),
			LGAC:          toStringPtr("Port Harcourt"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1I8IZLVCfu5ZX7rdQakeDN27XNA03Bkjb/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/12Jakf6t7AwrBp4q2HZoHGT_xZh_PmhJE/view?usp=sharing"),
			Lgas:          []string{"Abua/Odual", "Ahoada East", "Ahoada West", "Akuku-Toru", "Andoni", "Asari-Toru", "Bonny", "Degema", "Eleme", "Emohua", "Etche", "Gokana", "Ikwerre", "Khana", "Obio/Akpor", "Ogba/Egbema/Ndoni", "Ogu/Bolo", "Okrika", "Omuma", "Opobo/Nkoro", "Oyigbo", "Port Harcourt", "Tai"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Sokoto"),
			Governor:      toStringPtr("Ahmad Aliyu"),
			LGAC:          toStringPtr("Sokoto North"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1WpzqTbONg8AN2FpKHtggHNANq77sZqTN/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1qqYgCPKQg6uU1-aPPbDp-dR9njlYNLwl/view?usp=sharing"),
			Lgas:          []string{"Binji", "Bodinga", "Dange Shuni", "Gada", "Goronyo", "Gudu", "Gwadabawa", "Illela", "Isa", "Kebbe", "Kware", "Rabah", "Sabon Birni", "Shagari", "Silame", "Sokoto North", "Sokoto South", "Tambuwal", "Tangaza", "Tureta", "Wamako", "Wurno", "Yabo"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Taraba"),
			Governor:      toStringPtr("Agbu Kefas"),
			LGAC:          toStringPtr("Jalingo"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1vn-_gq-LgoHGH6BsO4UA-iHoEZ73D6qx/view?usp=sharing"),
			LgacImage:     toStringPtr("https://example.com/jalingo.jpg"),
			Lgas:          []string{"Ardo-Kola", "Bali", "Donga", "Gashaka", "Gassol", "Ibi", "Jalingo", "Karim Lamido", "Kurmi", "Lau", "Sardauna", "Takum", "Ussa", "Wukari", "Yorro", "Zing"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Yobe"),
			Governor:      toStringPtr("Mai Mala Buni"),
			LGAC:          toStringPtr("Damaturu"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1L-BwRqPyfjeod6Up5nUV8Kb86GN4X44p/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1TdULjESf2ZmHw_GJNzAfS6vs5mGO0STA/view?usp=sharing"),
			Lgas:          []string{"Bade", "Bursari", "Damaturu", "Fika", "Fune", "Geidam", "Gujba", "Gulani", "Jakusko", "Karasuwa", "Machina", "Nangere", "Nguru", "Potiskum", "Tarmuwa", "Yunusari", "Yusufari"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Zamfara"),
			Governor:      toStringPtr("Dauda Lawal"),
			LGAC:          toStringPtr("Gusau"),
			GovernorImage: toStringPtr("https://drive.google.com/file/d/1VqKBdJURgEaeb0A1TOQSRT-fxFbBq49p/view?usp=sharing"),
			LgacImage:     toStringPtr("https://drive.google.com/file/d/1bjjLA2zDjpU_ubsiPBdp0c48QNRefFL-/view?usp=sharing"),
			Lgas:          []string{"Anka", "Bakura", "Birnin Magaji/Kiyaw", "Bukkuyum", "Bungudu", "Gummi", "Gusau", "Kaura Namoda", "Maradun", "Maru", "Shinkafi", "Talata Mafara", "Tsafe", "Zurmi"},
		},
		{
			ID:            uuid.New(),
			State:         toStringPtr("Abuja (FCT)"),
			Governor:      toStringPtr("Nyesom Wike"), // FCT Minister, not Governor
			LGAC:          toStringPtr("Abuja Municipal"),
			GovernorImage: toStringPtr("https://example.com/wike.jpg"), // Placeholder, update as needed
			LgacImage:     toStringPtr("https://example.com/abuja.jpg"), // Placeholder, update as needed
			Lgas:          []string{"Abaji", "Abuja Municipal", "Bwari", "Gwagwalada", "Kuje", "Kwali"},
		},
	}

	for _, state := range states {
        // Serialize Lgas to JSON string
        lgasJSON, err := json.Marshal(state.Lgas)
        if err != nil {
            log.Printf("Failed to marshal LGAs for state %s: %v", *state.State, err)
            return err
        }

        // Check if state exists
        var existingState models.State
        result := db.Where("state = ?", *state.State).First(&existingState)
        if result.Error != nil {
            if result.Error == gorm.ErrRecordNotFound {
                // Create new record if not found
                if err := db.Create(&state).Error; err != nil {
                    log.Printf("Failed to create state %s: %v", *state.State, err)
                    return err
                }
                log.Printf("Created state %s with LGAs: %v", *state.State, state.Lgas)
            } else {
                log.Printf("Error querying state %s: %v", *state.State, result.Error)
                return result.Error
            }
        } else {
            // Update existing record with serialized LGAs
            if err := db.Model(&existingState).Updates(map[string]interface{}{
                "governor":       state.Governor,
                "lgac":           state.LGAC,
                "governor_image": state.GovernorImage,
                "lgac_image":     state.LgacImage,
                "lgas":           string(lgasJSON), // Pass as JSON string
            }).Error; err != nil {
                log.Printf("Failed to update state %s: %v", *state.State, err)
                return err
            }
            log.Printf("Updated state %s with LGAs: %v", *state.State, state.Lgas)
        }
    }

    return nil
}

func migrate(db *gorm.DB) error {
	// AutoMigrate all the models
	err := db.AutoMigrate(
		&models.User{},
		&models.Blacklist{},
		&models.IncidentReport{},
		&models.Media{},
		&models.Reward{},
		&models.Like{},
		&models.Notification{},
		&models.Comment{},
		&models.ReportType{},
		&models.IncidentReportUser{},
		&models.LGA{},
		&models.State{},
		&models.Bookmark{},
		&models.StateReportPercentage{},
		&models.MediaCount{},
		&models.LoginRequestMacAddress{},
		&models.UserImage{},
		&models.ReportCount{},
		&models.SubReport{},
		&models.Votes{},
		&models.UserPoints{},
		&models.Role{},
		&models.Post{},
		&models.ReportPostRequest{},
		&models.ReportUserRequest{},
		&models.Follow{},
		&models.OAuthState{},
		&models.Conversation{},
		&models.Message{},
	)
	
	if err != nil {
		return fmt.Errorf("migrations error: %v", err)
	}

	// Seed roles
	if err := SeedRoles(db); err != nil {
		return fmt.Errorf("seeding roles error: %v", err)
	}

	// Seed states
	if err := SeedStates(db); err != nil {
		return fmt.Errorf("seeding states error: %v", err)
	}

	// Add any additional migrations or seeds here if needed

	return nil
}