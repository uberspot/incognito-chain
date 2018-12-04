package zkp

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/ninjadotorg/constant/privacy-protocol"
)

// PKOneOfManyWitness is a protocol for Zero-knowledge Proof of Knowledge of one out of many commitments containing 0
// include Witness: CommitedValue, r []byte
type PKOneOfManyWitness struct {
	rand        *big.Int
	indexIsZero *int
	// general info
	commitments      []*privacy.EllipticPoint
	commitmentIndexs *[]*privacy.CMIndex
	index            byte
}

// PKOneOfManyProof contains Proof's value
type PKOneOfManyProof struct {
	cl, ca, cb, cd []*privacy.EllipticPoint
	f, za, zb      []*big.Int
	zd             *big.Int
	// general info
	commitments []*privacy.EllipticPoint
	index       byte
}

// Set sets Witness
func (wit *PKOneOfManyWitness) Set(
	commitments []*privacy.EllipticPoint,
	commitmentIndexs *[]*privacy.CMIndex,
	rand *big.Int,
	indexIsZero *int,
	index byte) {

	wit.commitmentIndexs = commitmentIndexs
	wit.commitments = commitments
	wit.indexIsZero = indexIsZero
	wit.rand = rand
	wit.index = index
}

// Set sets Proof
func (pro *PKOneOfManyProof) Set(
	commitments []*privacy.EllipticPoint,
	cl, ca, cb, cd []*privacy.EllipticPoint,
	f, za, zb []*big.Int,
	zd *big.Int,
	index byte) {

	pro.commitments = commitments
	pro.cl, pro.ca, pro.cb, pro.cd = cl, ca, cb, cd
	pro.f, pro.za, pro.zb = f, za, zb
	pro.zd = zd
	pro.index = index
}

func (pro *PKOneOfManyProof) Bytes() []byte {
	n := privacy.CMRingSizeExp
	N := privacy.CMRingSize
	var bytes []byte
	nBytes := 0

	// convert array cl to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, pro.cl[i].Compress()...)
		nBytes += privacy.CompressedPointSize
	}
	// convert array ca to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, pro.ca[i].Compress()...)
		nBytes += privacy.CompressedPointSize
	}

	// convert array cb to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, pro.cb[i].Compress()...)
		nBytes += privacy.CompressedPointSize
	}

	// convert array cd to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, pro.cd[i].Compress()...)
		nBytes += privacy.CompressedPointSize
	}

	// convert array f to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, pro.f[i].Bytes()...)
		nBytes += 32
	}

	// convert array za to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, pro.za[i].Bytes()...)
		nBytes += 32
	}

	// convert array zb to bytes array
	for i := 0; i < n; i++ {
		bytes = append(bytes, pro.zb[i].Bytes()...)
		nBytes += 32
	}

	// convert array zd to bytes array
	bytes = append(bytes, pro.zd.Bytes()...)
	nBytes += 32

	// get commitment's cmIndex
	cmIndex := make([]*privacy.CMIndex, N)
	for i := 0; i < N; i++ {
		cmIndex[i] = new(privacy.CMIndex)
		cmIndex[i].GetCmIndex(pro.commitments[i])

		// because length of cm index bytes is not specified
		// need to save length of cm index bytes

		cmIndexBytes := cmIndex[i].Bytes()
		bytes = append(bytes, byte(len(cmIndexBytes)))
		nBytes += 1
		bytes = append(bytes, cmIndexBytes...)
		nBytes += len(cmIndex[i].Bytes())
	}

	// append index
	bytes = append(bytes, pro.index)
	nBytes += 1

	fmt.Printf("Len of proof bytes: %v\n", nBytes)

	return bytes
}

func (pro *PKOneOfManyProof) SetBytes(bytes []byte) {
	n := privacy.CMRingSizeExp
	N := privacy.CMRingSize

	// get cl array
	pro.cl = make([]*privacy.EllipticPoint, n)
	for i := 0; i < n; i++ {
		pro.cl[i] = new(privacy.EllipticPoint)
		pro.cl[i], _ = privacy.DecompressKey(bytes[33*i: 33*i+33])
	}
	// get ca array
	pro.ca = make([]*privacy.EllipticPoint, n)
	for i := 0; i < n; i++ {
		pro.ca[i] = new(privacy.EllipticPoint)
		pro.ca[i], _ = privacy.DecompressKey(bytes[33*n+33*i: 33*n+33*i+33])
	}
	// get cb array
	pro.cb = make([]*privacy.EllipticPoint, n)
	for i := 0; i < n; i++ {
		pro.cb[i] = new(privacy.EllipticPoint)
		pro.cb[i], _ = privacy.DecompressKey(bytes[66*n+33*i: 66*n+33*i+33])
	}

	// get cd array
	pro.cd = make([]*privacy.EllipticPoint, n)
	for i := 0; i < n; i++ {
		pro.cd[i] = new(privacy.EllipticPoint)
		pro.cd[i], _ = privacy.DecompressKey(bytes[99*n+33*i: 99*n+33*i+33])
	}

	// get f array
	pro.f = make([]*big.Int, n)
	for i := 0; i < n; i++ {
		pro.f[i] = new(big.Int).SetBytes(bytes[4*n*33+32*i: 4*n*33+32*i+32])
	}

	// get za array
	pro.za = make([]*big.Int, n)
	for i := 0; i < n; i++ {
		pro.za[i] = new(big.Int).SetBytes(bytes[4*n*33+32*n+32*i: 4*n*33+32*n+32*i+32])
	}

	// get zb array
	pro.zb = make([]*big.Int, n)
	for i := 0; i < n; i++ {
		pro.zb[i] = new(big.Int).SetBytes(bytes[4*n*33+64*n+32*i: 4*n*33+64*n+32*i+32])
	}

	// get zd
	pro.zd = new(big.Int).SetBytes(bytes[4*n*33+3*n*32: 4*n*33+3*n*32+32])

	// get commitments list
	pro.commitments = make([]*privacy.EllipticPoint, N)
	lenPre := 4*n*33 + 3*n*32 + 32
	for i := 0; i < N; i++ {
		// get length of cm index
		cmIndexBytesLen := int(bytes[lenPre])
		cmIndex := new(privacy.CMIndex)
		cmIndex.SetBytes(bytes[lenPre+1: lenPre+1+cmIndexBytesLen])

		// for testing
		//rand := true
		//if i == 2 {
		//	rand = false
		//}
		pro.commitments[i] = cmIndex.GetCommitment()

		lenPre = lenPre + 1 + cmIndexBytesLen
	}

	//get index
	pro.index = bytes[len(bytes)-1]
	fmt.Printf("proof index setbytes: %v\n", pro.index)

}

// Prove creates proof for one out of many commitments containing 0
func (wit *PKOneOfManyWitness) Prove() (*PKOneOfManyProof, error) {
	// Check the number of Commitment list's elements

	N := len(wit.commitments)
	//temp := 1
	//n := 0
	//for temp < N {
	//	temp = temp << 1
	//	n++
	//}

	if N != privacy.CMRingSize {
		return nil, fmt.Errorf("the number of Commitment list's elements must be equal to CMRingSize")
	}

	n := privacy.CMRingSizeExp

	// Check indexIsZero
	if *wit.indexIsZero > N || *wit.indexIsZero < 0 {
		return nil, fmt.Errorf("Index is zero must be Index in list of commitments")
	}

	// Check Index
	if wit.index < privacy.SK || wit.index > privacy.RAND {
		return nil, fmt.Errorf("Index must be between index SK and index RAND")
	}

	// represent indexIsZero in binary
	indexIsZeroBinary := privacy.ConvertIntToBinary(*wit.indexIsZero, n)

	//
	r := make([]*big.Int, n)
	a := make([]*big.Int, n)
	s := make([]*big.Int, n)
	t := make([]*big.Int, n)
	u := make([]*big.Int, n)

	cl := make([]*privacy.EllipticPoint, n)
	ca := make([]*privacy.EllipticPoint, n)
	cb := make([]*privacy.EllipticPoint, n)
	cd := make([]*privacy.EllipticPoint, n)

	for j := n - 1; j >= 0; j-- {
		// Generate random numbers
		r[j], _ = rand.Int(rand.Reader, privacy.Curve.Params().N)
		a[j], _ = rand.Int(rand.Reader, privacy.Curve.Params().N)
		s[j], _ = rand.Int(rand.Reader, privacy.Curve.Params().N)
		t[j], _ = rand.Int(rand.Reader, privacy.Curve.Params().N)
		u[j], _ = rand.Int(rand.Reader, privacy.Curve.Params().N)

		// convert indexIsZeroBinary[j] to big.Int
		indexInt := big.NewInt(int64(indexIsZeroBinary[j]))

		// Calculate cl, ca, cb, cd
		// cl = Com(l, r)
		cl[j] = privacy.PedCom.CommitAtIndex(indexInt, r[j], wit.index)

		// ca = Com(a, s)
		ca[j] = privacy.PedCom.CommitAtIndex(a[j], s[j], wit.index)

		// cb = Com(la, t)
		la := new(big.Int)
		la.Mul(indexInt, a[j])
		la.Mod(la, privacy.Curve.Params().N)
		cb[j] = privacy.PedCom.CommitAtIndex(la, t[j], wit.index)
	}

	// Calculate: cd_k = ci^pi,k
	for k := 0; k < n; k++ {
		// Calculate pi,k which is coefficient of x^k in polynomial pi(x)
		res := &privacy.EllipticPoint{X: big.NewInt(0), Y: big.NewInt(0)}
		//tmp := privacy.EllipticPoint{X: big.NewInt(0), Y: big.NewInt(0)}

		for i := 0; i < N; i++ {
			iBinary := privacy.ConvertIntToBinary(i, n)
			pik := GetCoefficient(iBinary, k, n, a, indexIsZeroBinary)
			res = res.Add(wit.commitments[i].ScalarMul(pik))
		}

		comZero := privacy.PedCom.CommitAtIndex(big.NewInt(0), u[k], wit.index)
		res = res.Add(comZero)
		cd[k] = res
	}

	// Calculate x
	x := big.NewInt(0)

	for j := 0; j <= n-1; j++ {
		*x = *GenerateChallengeFromByte([][]byte{x.Bytes(), cl[j].Compress(), ca[j].Compress(), cb[j].Compress(), cd[j].Compress()})
		x.Mod(x, privacy.Curve.Params().N)
	}

	// Calculate za, zb zd
	za := make([]*big.Int, n)
	zb := make([]*big.Int, n)
	zd := new(big.Int)
	f := make([]*big.Int, n)

	for j := n - 1; j >= 0; j-- {
		// f = lx + a
		f[j] = new(big.Int)
		f[j] = f[j].Mul(big.NewInt(int64(indexIsZeroBinary[j])), x)
		f[j].Add(f[j], a[j])
		f[j].Mod(f[j], privacy.Curve.Params().N)

		// za = s + rx
		za[j] = new(big.Int)
		za[j].Mul(r[j], x)
		za[j].Add(za[j], s[j])
		za[j].Mod(za[j], privacy.Curve.Params().N)

		// zb = r(x - f) + t
		zb[j] = new(big.Int)
		zb[j].Sub(x, f[j])
		zb[j].Mod(zb[j], privacy.Curve.Params().N)
		zb[j].Mul(zb[j], r[j])
		zb[j].Add(zb[j], t[j])
		zb[j].Mod(zb[j], privacy.Curve.Params().N)
	}

	// zdInt := big.NewInt(0)
	zd.Exp(x, big.NewInt(int64(n)), privacy.Curve.Params().N)
	zd.Mul(zd, wit.rand)
	// zdInt.Mul(zdInt, new(big.Int).SetBytes(rand))

	uxInt := big.NewInt(0)
	sumInt := big.NewInt(0)
	for k := 0; k < n; k++ {
		uxInt.Exp(x, big.NewInt(int64(k)), privacy.Curve.Params().N)
		uxInt.Mul(uxInt, u[k])
		sumInt.Add(sumInt, uxInt)
		sumInt.Mod(sumInt, privacy.Curve.Params().N)
	}

	sumInt.Sub(privacy.Curve.Params().N, sumInt)

	zd.Add(zd, sumInt)
	zd.Mod(zd, privacy.Curve.Params().N)
	var proof PKOneOfManyProof
	proof.Set(wit.commitments, cl, ca, cb, cd, f, za, zb, zd, wit.index)

	return &proof, nil
}

func (pro *PKOneOfManyProof) Verify() bool {
	N := privacy.CMRingSize
	n := privacy.CMRingSizeExp

	//temp := 1
	//n := 0
	//for temp < N {
	//	temp = temp << 1
	//	n++
	//}
	// Calculate x
	x := big.NewInt(0)

	for j := 0; j <= n-1; j++ {
		*x = *GenerateChallengeFromByte([][]byte{x.Bytes(), pro.cl[j].Compress(), pro.ca[j].Compress(), pro.cb[j].Compress(), pro.cd[j].Compress()})
		x.Mod(x, privacy.Curve.Params().N)
	}

	for i := 0; i < n; i++ {
		// Check cl^x * ca = Com(f, za)
		//leftPoint1 := new(privacy.EllipticPoint)
		//leftPoint1.X, leftPoint1.Y = privacy.Curve.ScalarMult(pro.cl[i].X, pro.cl[i].Y, x.Bytes())
		//leftPoint1.X, leftPoint1.Y = privacy.Curve.Add(leftPoint1.X, leftPoint1.Y, pro.ca[i].X, pro.ca[i].Y)
		leftPoint1 := pro.cl[i].ScalarMul(x).Add(pro.ca[i])

		rightPoint1 := privacy.PedCom.CommitAtIndex(pro.f[i], pro.za[i], pro.index)
		fmt.Printf("Left point 1 X: %v\n", leftPoint1.X)
		fmt.Printf("Right point 1 X: %v\n", rightPoint1.X)
		fmt.Printf("Left point 1 Y: %v\n", leftPoint1.Y)
		fmt.Printf("Right point 1 Y: %v\n", rightPoint1.Y)

		if !leftPoint1.IsEqual(rightPoint1) {
			return false
		}

		// Check cl^(x-f) * cb = Com(0, zb)

		xSubF := new(big.Int)
		xSubF.Sub(x, pro.f[i])
		xSubF.Mod(xSubF, privacy.Curve.Params().N)

		leftPoint2 := pro.cl[i].ScalarMul(xSubF).Add(pro.cb[i])
		//	new(privacy.EllipticPoint)
		//leftPoint2.X, leftPoint2.Y = privacy.Curve.ScalarMult(pro.cl[i].X, pro.cl[i].Y, xSubF.Bytes())
		//leftPoint2.X, leftPoint2.Y = privacy.Curve.Add(leftPoint2.X, leftPoint2.Y, pro.cb[i].X, pro.cb[i].Y)
		rightPoint2 := privacy.PedCom.CommitAtIndex(big.NewInt(0), pro.zb[i], pro.index)

		//fmt.Printf("Left point 2 X: %v\n", leftPoint2.X)
		//fmt.Printf("Right point 2 X: %v\n", rightPoint2.X)
		//fmt.Printf("Left point 2 Y: %v\n", leftPoint2.Y)
		//fmt.Printf("Right point 2 Y: %v\n", rightPoint2.Y)

		if !leftPoint2.IsEqual(rightPoint2) {
			return false
		}
	}

	leftPoint3 := privacy.EllipticPoint{X: big.NewInt(0), Y: big.NewInt(0)}
	leftPoint32 := privacy.EllipticPoint{X: big.NewInt(0), Y: big.NewInt(0)}
	tmpPoint := new(privacy.EllipticPoint)

	for i := 0; i < N; i++ {
		iBinary := privacy.ConvertIntToBinary(i, n)

		exp := big.NewInt(1)
		fji := big.NewInt(1)
		for j := n - 1; j >= 0; j-- {
			if iBinary[j] == 1 {
				fji.Set(pro.f[j])
			} else {
				fji.Sub(x, pro.f[j])
				fji.Mod(fji, privacy.Curve.Params().N)
			}

			exp.Mul(exp, fji)
			exp.Mod(exp, privacy.Curve.Params().N)
		}

		tmpPoint.X, tmpPoint.Y = privacy.Curve.ScalarMult(pro.commitments[i].X, pro.commitments[i].Y, exp.Bytes())
		leftPoint3.X, leftPoint3.Y = privacy.Curve.Add(leftPoint3.X, leftPoint3.Y, tmpPoint.X, tmpPoint.Y)
	}

	for k := 0; k < n; k++ {
		xk := big.NewInt(0)
		xk.Exp(x, big.NewInt(int64(k)), privacy.Curve.Params().N)

		xk.Sub(privacy.Curve.Params().N, xk)

		tmpPoint.X, tmpPoint.Y = privacy.Curve.ScalarMult(pro.cd[k].X, pro.cd[k].Y, xk.Bytes())
		leftPoint32.X, leftPoint32.Y = privacy.Curve.Add(leftPoint32.X, leftPoint32.Y, tmpPoint.X, tmpPoint.Y)
	}

	leftPoint3.X, leftPoint3.Y = privacy.Curve.Add(leftPoint3.X, leftPoint3.Y, leftPoint32.X, leftPoint32.Y)

	rightPoint3 := privacy.PedCom.CommitAtIndex(big.NewInt(0), pro.zd, pro.index)

	fmt.Printf("Left point 3 X: %v\n", leftPoint3.X)
	fmt.Printf("Right point 3 X: %v\n", rightPoint3.X)
	fmt.Printf("Left point 3 Y: %v\n", leftPoint3.Y)
	fmt.Printf("Right point 3 Y: %v\n", rightPoint3.Y)
	if leftPoint3.X.Cmp(rightPoint3.X) != 0 || leftPoint3.Y.Cmp(rightPoint3.Y) != 0 {
		return false
	}

	return true
}

//TestPKOneOfMany test protocol for one of many Commitment is Commitment to zero
func TestPKOneOfMany() bool {
	witness := new(PKOneOfManyWitness)

	indexIsZero := 2

	// list of commitments
	commitments := make([]*privacy.EllipticPoint, privacy.CMRingSize)
	SNDerivators := make([]*big.Int, privacy.CMRingSize)
	randoms := make([]*big.Int, privacy.CMRingSize)

	for i := 0; i < privacy.CMRingSize; i++ {
		SNDerivators[i] = privacy.RandInt()
		randoms[i] = privacy.RandInt()
		commitments[i] = privacy.PedCom.CommitAtIndex(SNDerivators[i], randoms[i], privacy.SND)
	}

	// create Commitment to zero at indexIsZero
	SNDerivators[indexIsZero] = big.NewInt(0)
	commitments[indexIsZero] = privacy.PedCom.CommitAtIndex(SNDerivators[indexIsZero], randoms[indexIsZero], privacy.SND)

	witness.Set(commitments, nil, randoms[indexIsZero], &indexIsZero, privacy.SND)
	//start := time.Now()
	proof, err := witness.Prove()

	// Convert proof to bytes array
	proofBytes := proof.Bytes()
	fmt.Printf("Proof bytes: %v\n", proofBytes)
	fmt.Printf("Proof bytes len: %v\n", len(proofBytes))

	// revert bytes array to proof
	proof2 := new(PKOneOfManyProof)
	proof2.SetBytes(proofBytes)

	if err != nil {
		fmt.Println(err)
	}
	res := proof.Verify()

	//end := time.Now()
	//fmt.Printf("%v_+_\n", end.Sub(start))
	fmt.Println(res)
	return res
}

//// Get coefficient of x^k in polynomial pi(x)
func GetCoefficient(iBinary []byte, k int, n int, a []*big.Int, l []byte) *big.Int {
	res := privacy.Poly{big.NewInt(1)}
	var fji privacy.Poly

	for j := n - 1; j >= 0; j-- {
		// fj := privacy.Poly{new(big.Int).SetBytes(a[j]), big.NewInt(int64(l[j]))}
		fj := privacy.Poly{a[j], big.NewInt(int64(l[j]))}
		if iBinary[j] == 0 {
			fji = privacy.Poly{big.NewInt(0), big.NewInt(1)}.Sub(fj, privacy.Curve.Params().N)
		} else {
			fji = fj
		}
		res = res.Mul(fji, privacy.Curve.Params().N)
	}

	if res.GetDegree() < k {
		return big.NewInt(0)
	}
	return res[k]
}
