module github.com/MobRulesGames/haunts

go 1.23.0

require (
	code.google.com/p/freetype-go v0.0.0-20120725121025-28cc5fbc5d0b
	github.com/MobRulesGames/GoLLRB v0.0.0-20121115013357-10dddd6fc70e
	github.com/MobRulesGames/fmod v0.0.0-20121207023041-90f897047d59
	github.com/MobRulesGames/fsnotify v0.0.0-20121110053322-1b2bc1227408
	github.com/MobRulesGames/golua v0.0.0-00010101000000-000000000000
	github.com/MobRulesGames/mathgl v0.0.0-20120424214601-79bd4ce3042d
	github.com/MobRulesGames/memory v0.0.0-20120626004817-db5bb35fd894
	github.com/go-gl-legacy/gl v0.0.0-20150223033340-df25b1fe668d
	github.com/go-gl-legacy/glu v0.0.0-20150315173544-b54aa06bc77a
	github.com/runningwild/glop v0.0.0-20130331194942-bcbcf4982510
	github.com/smartystreets/goconvey v1.8.1
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/howeyc/fsnotify v0.9.0 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/runningwild/yedparse v0.0.0-20120306014153-f7df1db2f9d9 // indirect
	github.com/smarty/assertions v1.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace code.google.com/p/freetype-go => github.com/golang/freetype v0.0.0-20120725121025-28cc5fbc5d0b

replace github.com/MobRulesGames/mathgl => github.com/caffeine-storm/mathgl v0.0.0-20250304142043-9a68bb7bb47a

replace github.com/runningwild/glop => github.com/caffeine-storm/glop v0.0.0-20250422182037-10ef83c0f74c

replace github.com/go-gl-legacy/gl => github.com/caffeine-storm/gl v0.0.0-20240909160157-d1b38f2deb16

replace github.com/go-gl-legacy/glu => github.com/caffeine-storm/glu v0.0.0-20240828152149-38a5ac65629c

replace github.com/MobRulesGames/golua => github.com/caffeine-storm/golua v0.0.0-20240910150920-bb0104c032e4
