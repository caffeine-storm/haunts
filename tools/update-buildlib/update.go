package updatebuildlib

type Tree struct {
	data []byte
}

func MakeTreeFromTarball(tarball []byte) *Tree {
	return &Tree{
		data: tarball,
	}
}

func (t *Tree) Matches(other *Tree) bool {
	return false
}
