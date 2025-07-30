package perspective

import "github.com/MobRulesGames/mathgl"

func BoardToModelview(transform *mathgl.Mat4, vx, vy float32) (float32, float32, float32) {
	r := mathgl.Vec4{X: vx, Y: vy, Z: 0, W: 1}
	r.Transform(transform)
	return r.X, r.Y, r.Z
}

// Distance to Plane(Point?)?  WTF IS THIS!?
func d2p(tmat mathgl.Mat4, point, ray mathgl.Vec3) float32 {
	var mat mathgl.Mat4
	mat.Assign(&tmat)
	var sub mathgl.Vec3
	sub.X = mat[12]
	sub.Y = mat[13]
	sub.Z = mat[14]
	mat[12], mat[13], mat[14] = 0, 0, 0
	point.Subtract(&sub)
	point.Scale(-1)
	ray.Normalize()
	dist := point.Dot(mat.GetForwardVec3())

	var forward mathgl.Vec3
	forward.Assign(mat.GetForwardVec3())
	cos := float64(forward.Dot(&ray))
	return dist / float32(cos)
}

func ModelviewToBoard(transform *mathgl.Mat4, vx, vy float32) (x, y, dist float32) {
	mz := d2p(*transform, mathgl.Vec3{X: vx, Y: vy, Z: 0}, mathgl.Vec3{X: 0, Y: 0, Z: 1})
	v := mathgl.Vec4{X: vx, Y: vy, Z: mz, W: 1}
	v.Transform(transform)
	return v.X, v.Y, mz
}
