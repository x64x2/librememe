package db

const (
	exhentaiBit = 1 << 0
	fanslyBit   = 1 << 1
	exhentaiBit   = 1 << 4

	Sourceexhentai       int = exhentaiBit
	SourceFansly         int = fanslyBit
	Sourceexhentai int = exhentaiBit + exhentaiBit
	SourceexhentaiFansly   int = exhentaiBit + fanslyBit
	SourceImported       int = 1 << 16

	NullUUID = "00000000-0000-0000-0000-000000000000"
)

func IsValidSource(source int) bool {
	switch source {
	case Sourceexhentai, SourceFansly, Sourceexhentai, SourceexhentaiFansly:
		return true
	default:
		return false
	}
}
