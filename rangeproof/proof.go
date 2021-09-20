package rangeproof

import (
	"fmt"
	"strconv"

	"github.com/privacybydesign/gabi/big"
	"github.com/privacybydesign/gabi/gabikeys"
	"github.com/privacybydesign/gabi/internal/common"
	"github.com/privacybydesign/gabi/zkproof"

	"github.com/go-errors/errors"
)

/*
This subpackage of gabi implements a variation of the inequality/range proof protocol given in
section 6.2.6/6.3.6 of "Specification of the Identity Mixer Cryptographic Library Version 2.3.0".

Specifically, given an attribute m; a value k; a positive number a; and a positive or negative sign
(i.e. 1 or -1), this subpackage allows clients to prove that sign*(a*m - k) >= 0, by writing the
left hand side as a sum of squares, and then proving knowledge of the square roots of these squares.
From this, the verifier can infer that sign*(a*m - k) must be non-negative, i.e. a*m >= k or a*m <=
k if sign = 1 or -1 respectively.

The following changes were made with respect to the Identity Mixer specification:
- There is no direct support for the > and < operators: the end user should do boundary adjustment
  for this themselves.
- There is support for sum of 3 squares as well as sum of 4 squares, for space optimization.
- There is no separate commitment to the difference between bound and attribute value.

This results in that our code proves the following substatements:

    C_i = R^(d_i) S^(v_i)
    R^(sign*k) \product_i C_i^(d_i) = R^(sign*a*m) S^(v_5)

where
- m is the attribute value,
- k, a and sign are fixed constants specified in the proof structure,
- d_i are values such that sign*(a*m - k) = \sum_i (d_i)^2,
- v_i are computational hiders for the d_i,
- v_5 = \sum_i d_i * v_i.

The proof of soundness for this protocol is relatively straightforward, but as we are not aware of
its occurence in literature, we provide it here for completeness.

---

First note that we can assume a=1 without loss of generality. Then we define Adversary A as a
probabilistic polynomial-time algorithm that acts as follows:
- it takes as starting input S, R, Z, n, and k,
- it participates as the receiver in the issuance protocol to obtain a CL signature on m < k,
- it participates as prover in the proving protocol showing a CL signature on m', as well as
  providing bases C1, C2, C3 and proving knowledge of d1, d2, d3, v1, v2, v3, v5 such that
   * C1 = R^d1 S^v1
   * C2 = R^d2 S^v2
   * C3 = R^d3 S^v3
   * C1^d1 C2^d2 C3^d3 R^k = R^m' S^v5,
  with success probability at least epsilon, where epsilon is a non-neglible function of log(n).

Theorem: the existence of Adversary A contradicts the strong RSA assumption.

Proof: From A we can derive two probabilistic polynomial-time algorithms F and G

- F: Run A, then rewind to extract the full CL signature on m'. Then return that signature if
  m != m', fail otherwise
- G: Run A, then rewind to extract m', d1, d2, d3, v1, v2, v3, v5. Then return m, d1, d2, d3,
  v1, v2, v3, v5 iff m == m', fail otherwise

By construction at least one of F, G will succeed with probability at least epsilon/2.

By theorem 1 of "A Signature Scheme with Efficient Protocols" by Camenisch and Lysyanskaya (CL03),
the existence of F with non-neglible success probability contradicts the strong RSA assumption.

Next, let us show that existence of G with non-neglible success probability also contradicts the
strong RSA assumption. Let (n, u) be a flexible RSA problem. We choose random prime e > 4, random
integers r1, r2, k, and m>k, and random element v in QRn. We then let
- S = u^e
- R = S^r1
- Z = S^r2,
and present S,R,Z,n as public key to the adversary, together with k.

We then use our knowledge of the e-th roots of S, R, and Z, together with our ability to extract v'
from G to issue the signature on m similar to the approach taken in the proofs of lemmas 3-5 in
CL03. Next, we provide k and receive back m, d1, d2, d3, v1, v2, v3 and v5, for which it holds that
R^(d1^2+d2^2+d3^2+k-m) = S^(v5-d1*v1+d2*v2+d3*v3).

Since k > m, and since d1, d2, d3 are real, we have d1^2+d2^2+d3^2+k-m > 0. Now either phi(n)
divides d1^2+d2^2+d3^2+k-m, or v5-d1*v1+d2*v2+d3*v3 is not zero. If phi(n) divides
d1^2+d2^2+d3^2+k-m, then by Lemma 11 in CL03, we can factor n and trivially solve the instance.

Otherwise, we now have nonzero a = d1^2+d2^2+d3^2+k-m, and b = v5-d1*v1+d2*v2+d3*v3, such that
R^a = S^b. Since e does not divide phi(n) (otherwise, we could just factor n again, this time by
lemma 12 in CL03), this also implies u^(r1*a) = u^b. Since a and b are bounded (each of their
components is proven to be smaller than a bound during the ZKP), we can take r1 large enough to
guarantee r1*a > b. Then r1*a-b > 0, and by multiplication by u^-b, we get u^(r1*a-b) = 1, hence
phi(n) | r1*a-b, which means we can factor n and use that to solve the flexible RSA problem.

This shows that the existence of algorithm G with non-neglible success probability also contradicts
strong RSA, hence the existence of algorithm A contradicts the strong RSA assumption.

Notes:
- The techniques used to fake issuance can be generalized to multiple issuances using techiques
  similar to those used in CL03 for lemmas 3-5; the rest of the proof then needs e to be replaced
  with E, the product of all used e_i's.
- This proof nowhere uses that the amount of squares we use is three; hence it also works when using
  four squares.
*/

type (
	// Statement states that an attribute m satisfies Sign*(Factor*m-Bound) >= 0, and that
	// Sign*(Factor*m-Bound) can be split into squares with the given Splitter. E.g. if Factor = 1
	// then Factor*m >= Bound. Defaults to four square splitter when splitter is not specified.
	Statement struct {
		Sign     int
		Factor   uint
		Bound    *big.Int
		Splitter SquareSplitter
	}

	StatementType int

	ProofStructure struct {
		cRep     []zkproof.QrRepresentationProofStructure
		mCorrect zkproof.QrRepresentationProofStructure

		index int
		sign  int
		a     uint
		k     *big.Int

		splitter SquareSplitter
		ld       uint
	}

	Proof struct {
		// Actual proof responses
		Cs         []*big.Int `json:"Cs"`
		DResponses []*big.Int `json:"ds"`
		VResponses []*big.Int `json:"vs"`
		V5Response *big.Int   `json:"v5"`
		MResponse  *big.Int   `json:"-"`

		// Proof structure description
		Ld   uint     `json:"l_d"`
		Sign int      `json:"sign"`
		A    uint     `json:"a"`
		K    *big.Int `json:"k"`
	}

	ProofCommit struct {
		// Bases
		c []*big.Int

		// Secrets
		d            []*big.Int
		dRandomizers []*big.Int
		v            []*big.Int
		vRandomizers []*big.Int
		v5           *big.Int
		v5Randomizer *big.Int
		m            *big.Int
		mRandomizer  *big.Int
	}

	proof       Proof
	proofCommit ProofCommit
)

const (
	GreaterOrEqual StatementType = iota
	LesserOrEqual
)

var (
	ErrFalseStatement  = errors.New("requested inequality does not hold")
	ErrUnsupportedSign = errors.New("unsupported sign: must be 1 or -1")
)

func NewStatement(typ StatementType, bound *big.Int) (*Statement, error) {
	sign, err := typ.Sign()
	if err != nil {
		return nil, err
	}
	return &Statement{Sign: sign, Factor: 1, Bound: new(big.Int).Set(bound)}, nil
}

// Create a new proof structure for proving a statement of the form sign(factor*m - bound) >= 0.
//
// index specifies the index of the attribute.
// splitter describes the method used for splitting numbers into sum of squares.
func NewProofStructure(index, sign int, factor uint, bound *big.Int, splitter SquareSplitter) (*ProofStructure, error) {
	if splitter == nil {
		splitter = &FourSquaresSplitter{}
	}

	if splitter.SquareCount() == 3 {
		if factor != 1 {
			return nil, errors.New("factor must be 1")
		}
		// Not all numbers can be written as sum of 3 squares, but n for which n == 2 (mod 4) can
		// so ensure that factor*m-bound falls into that category
		factor *= 4
		bound = new(big.Int).Mul(bound, big.NewInt(4)) // ensure we dont overwrite callers copy of bound
		bound.Sub(bound, big.NewInt(2))
	}

	return newWithParams(index, sign, factor, bound, splitter, splitter.SquareCount(), splitter.Ld())
}

func newWithParams(index, sign int, a uint, k *big.Int, split SquareSplitter, nSplit int, ld uint) (*ProofStructure, error) {
	if nSplit > 4 {
		return nil, errors.New("no support for range proofs with delta split in more than 4 squares")
	}
	if sign != 1 && sign != -1 {
		return nil, ErrUnsupportedSign
	}

	var exp *big.Int
	if sign == 1 {
		exp = new(big.Int).Neg(k)
	} else {
		exp = new(big.Int).Set(k)
	}
	result := &ProofStructure{
		mCorrect: zkproof.QrRepresentationProofStructure{
			Lhs: []zkproof.LhsContribution{
				{Base: fmt.Sprintf("R%d", index), Power: exp},
			},
			Rhs: []zkproof.RhsContribution{
				{Base: "S", Secret: "v5", Power: -1},
				{Base: fmt.Sprintf("R%d", index), Secret: "m", Power: -int64(a) * int64(sign)},
			},
		},

		index: index,
		sign:  sign,
		a:     a,
		k:     new(big.Int).Set(k),

		splitter: split,
		ld:       ld,
	}

	for i := 0; i < nSplit; i++ {
		result.cRep = append(result.cRep, zkproof.QrRepresentationProofStructure{
			Lhs: []zkproof.LhsContribution{
				{Base: fmt.Sprintf("C%d", i), Power: big.NewInt(1)},
			},
			Rhs: []zkproof.RhsContribution{
				{Base: fmt.Sprintf("R%d", index), Secret: fmt.Sprintf("d%d", i), Power: 1},
				{Base: "S", Secret: fmt.Sprintf("v%d", i), Power: 1},
			},
		})

		result.mCorrect.Rhs = append(result.mCorrect.Rhs, zkproof.RhsContribution{
			Base:   fmt.Sprintf("C%d", i),
			Secret: fmt.Sprintf("d%d", i),
			Power:  1,
		})
	}

	return result, nil
}

func (statement *Statement) ProofStructure(index int) (*ProofStructure, error) {
	return NewProofStructure(index, statement.Sign, statement.Factor, statement.Bound, statement.Splitter)
}

func (typ StatementType) Sign() (int, error) {
	switch typ {
	case GreaterOrEqual:
		return 1, nil
	case LesserOrEqual:
		return -1, nil
	default:
		return 0, ErrUnsupportedSign
	}
}

func (s *ProofStructure) CommitmentsFromSecrets(g *gabikeys.PublicKey, m, mRandomizer *big.Int) ([]*big.Int, *ProofCommit, error) {
	var err error

	d := new(big.Int).Mul(m, big.NewInt(int64(s.a)))
	d.Sub(d, s.k)
	if s.sign == -1 {
		d.Neg(d)
	}

	if d.Sign() < 0 {
		return nil, nil, ErrFalseStatement
	}

	commit := &proofCommit{
		m:           m,
		mRandomizer: mRandomizer,
	}

	commit.d, err = s.splitter.Split(d)
	if err != nil {
		return nil, nil, err
	}
	if len(commit.d) != len(s.cRep) {
		return nil, nil, errors.New("split function returned wrong number of results")
	}

	// Check d values and generate randomizers for them
	commit.dRandomizers = make([]*big.Int, len(commit.d))
	for i, v := range commit.d {
		if v.BitLen() > int(s.ld) {
			return nil, nil, errors.New("split function returned oversized d")
		}
		commit.dRandomizers[i], err = common.RandomBigInt(s.ld + g.Params.Lh + g.Params.Lstatzk)
		if err != nil {
			return nil, nil, err
		}
	}

	// Generate v and vRandomizers
	commit.v = make([]*big.Int, len(commit.d))
	commit.vRandomizers = make([]*big.Int, len(commit.d))
	for i := range commit.d {
		commit.v[i], err = common.RandomBigInt(g.Params.Lm)
		if err != nil {
			return nil, nil, err
		}
		commit.vRandomizers[i], err = common.RandomBigInt(g.Params.Lm + g.Params.Lh + g.Params.Lstatzk)
		if err != nil {
			return nil, nil, err
		}
	}

	// Generate v5 and its randomizer
	commit.v5 = big.NewInt(0)
	for i := range commit.d {
		contrib := new(big.Int).Mul(commit.d[i], commit.v[i])
		commit.v5.Add(commit.v5, contrib)
	}
	commit.v5Randomizer, err = common.RandomBigInt(g.Params.Lm + s.ld + 2 + g.Params.Lh + g.Params.Lstatzk)
	if err != nil {
		return nil, nil, err
	}

	// Calculate the bases
	commit.c = make([]*big.Int, len(commit.d))
	for i := range commit.d {
		commit.c[i] = new(big.Int).Exp(g.R[s.index], commit.d[i], g.N)
		commit.c[i].Mul(commit.c[i], new(big.Int).Exp(g.S, commit.v[i], g.N))
		commit.c[i].Mod(commit.c[i], g.N)
	}

	bases := zkproof.NewBaseMerge(g, commit)

	var contributions []*big.Int
	contributions = s.mCorrect.CommitmentsFromSecrets(g, contributions, &bases, commit)
	for i := range commit.d {
		contributions = s.cRep[i].CommitmentsFromSecrets(g, contributions, &bases, commit)
	}

	return contributions, (*ProofCommit)(commit), nil
}

func (s *ProofStructure) BuildProof(commit *ProofCommit, challenge *big.Int) *Proof {
	result := &Proof{
		Cs:         make([]*big.Int, len(commit.c)),
		DResponses: make([]*big.Int, len(commit.d)),
		VResponses: make([]*big.Int, len(commit.v)),
		V5Response: new(big.Int).Add(new(big.Int).Mul(challenge, commit.v5), commit.v5Randomizer),
		MResponse:  new(big.Int).Add(new(big.Int).Mul(challenge, commit.m), commit.mRandomizer),

		Ld:   s.ld,
		Sign: s.sign,
		A:    s.a,
		K:    new(big.Int).Set(s.k),
	}

	for i := range commit.c {
		result.Cs[i] = new(big.Int).Set(commit.c[i])
	}
	for i := range commit.d {
		result.DResponses[i] = new(big.Int).Add(new(big.Int).Mul(challenge, commit.d[i]), commit.dRandomizers[i])
	}
	for i := range commit.v {
		result.VResponses[i] = new(big.Int).Add(new(big.Int).Mul(challenge, commit.v[i]), commit.vRandomizers[i])
	}

	return result
}

func (s *ProofStructure) VerifyProofStructure(g *gabikeys.PublicKey, p *Proof) bool {
	if len(s.cRep) != len(p.Cs) || len(s.cRep) != len(p.DResponses) || len(s.cRep) != len(p.VResponses) {
		return false
	}

	if p.V5Response == nil || p.MResponse == nil {
		return false
	}

	if uint(p.V5Response.BitLen()) > g.Params.Lm+s.ld+2+g.Params.Lh+g.Params.Lstatzk+1 ||
		uint(p.MResponse.BitLen()) > g.Params.Lm+g.Params.Lh+g.Params.Lstatzk+1 {
		return false
	}

	for i := range s.cRep {
		if p.Cs[i] == nil || p.DResponses[i] == nil || p.VResponses[i] == nil {
			return false
		}

		if p.Cs[i].BitLen() > g.N.BitLen() ||
			uint(p.DResponses[i].BitLen()) > s.ld+g.Params.Lh+g.Params.Lstatzk+1 ||
			uint(p.VResponses[i].BitLen()) > g.Params.Lm+g.Params.Lh+g.Params.Lstatzk+1 {
			return false
		}
	}

	return true
}

func (s *ProofStructure) CommitmentsFromProof(g *gabikeys.PublicKey, p *Proof, challenge *big.Int) []*big.Int {
	bases := zkproof.NewBaseMerge(g, (*proof)(p))

	var contributions []*big.Int
	contributions = s.mCorrect.CommitmentsFromProof(g, contributions, challenge, &bases, (*proof)(p))
	for i := range s.cRep {
		contributions = s.cRep[i].CommitmentsFromProof(g, contributions, challenge, &bases, (*proof)(p))
	}

	return contributions
}

// ProvesStatement returns whether the Proof proves or implies the specified statement.
func (p *Proof) ProvesStatement(sign int, factor uint, bound *big.Int) bool {
	if sign != 1 && sign != -1 {
		return false
	}
	if len(p.Cs) == 3 {
		factor *= 4
		bound = new(big.Int).Mul(bound, big.NewInt(4))
		bound.Sub(bound, big.NewInt(2))
	}
	return p.Sign == sign && p.A == factor &&
		(p.K.Cmp(bound) == 0 || p.K.Cmp(bound) == sign)
}

// Proves returns whether the Proof proves or implies the specified statement.
func (p *Proof) Proves(statement *Statement) bool {
	return p.ProvesStatement(statement.Sign, statement.Factor, statement.Bound)
}

// ProvenStatement returns the statement that this proof proves. Calling the second and third return
// parameters "factor" and "bound" respectively, then
//    factor*attribute - bound >= 0  or  <= 0
// where the inequality type is returned as the first parameter.
//
// NB: this method does not verify the proof. Do not trust the output unless proof.Verify() has been
// invoked first.
func (p *Proof) ProvenStatement() (StatementType, uint, *big.Int) {
	bound := new(big.Int).Set(p.K)
	factor := p.A
	if len(p.Cs) == 3 {
		bound.Add(bound, big.NewInt(2)).Rsh(bound, 2)
		factor >>= 2
	}
	var typ StatementType
	switch p.Sign {
	case 1:
		typ = GreaterOrEqual
	case -1:
		typ = LesserOrEqual
	}
	return typ, factor, bound
}

// Extract proof structure from proof
func (p *Proof) ExtractStructure(index int, g *gabikeys.PublicKey) (*ProofStructure, error) {
	// Check that all values needed for the structure are present and reasonable
	//
	// ld > lm is never reasonable since that implies a difference greater than 2^(2*lm)
	//  which is bigger than m*a
	// p.K >= 2^lm+sizeof(a) is never reasonable since that makes |m*a| < |k|, making
	//  the proof statement trivial (it either always or never holds)
	if p.K == nil || p.Ld > g.Params.Lm || len(p.Cs) < 3 || len(p.Cs) > 4 ||
		p.K.BitLen() > int(g.Params.Lm+strconv.IntSize) ||
		(len(p.Cs) == 3 && p.A != 4) {
		return nil, errors.New("invalid proof")
	}
	return newWithParams(index, p.Sign, p.A, p.K, nil, len(p.Cs), p.Ld)
}

// ---
// Commit structure keyproof interfaces
// ---
func (c *proofCommit) Secret(name string) *big.Int {
	if name == "m" {
		return c.m
	}
	if name == "v5" {
		return c.v5
	}
	if name[0] == 'v' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(c.v) {
			return nil
		}
		return c.v[i]
	}
	if name[0] == 'd' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(c.d) {
			return nil
		}
		return c.d[i]
	}
	return nil
}

func (c *proofCommit) Randomizer(name string) *big.Int {
	if name == "m" {
		return c.mRandomizer
	}
	if name == "v5" {
		return c.v5Randomizer
	}
	if name[0] == 'v' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(c.vRandomizers) {
			return nil
		}
		return c.vRandomizers[i]
	}
	if name[0] == 'd' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(c.dRandomizers) {
			return nil
		}
		return c.dRandomizers[i]
	}
	return nil
}

func (c *proofCommit) Base(name string) *big.Int {
	if name[0] == 'C' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(c.c) {
			return nil
		}
		return c.c[i]
	}
	return nil
}

func (c *proofCommit) Exp(ret *big.Int, name string, exp, n *big.Int) bool {
	base := c.Base(name)
	if base == nil {
		return false
	}
	ret.Exp(base, exp, n)
	return true
}

func (c *proofCommit) Names() []string {
	result := make([]string, 0, len(c.c))
	for i := range c.c {
		result = append(result, fmt.Sprintf("C%d", i))
	}

	return result
}

// ---
// Proof structure keyproof interfaces
// ---
func (p *proof) ProofResult(name string) *big.Int {
	if name == "m" {
		return p.MResponse
	}
	if name == "v5" {
		return p.V5Response
	}
	if name[0] == 'v' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(p.VResponses) {
			return nil
		}
		return p.VResponses[i]
	}
	if name[0] == 'd' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(p.DResponses) {
			return nil
		}
		return p.DResponses[i]
	}
	return nil
}

func (p *proof) Base(name string) *big.Int {
	if name[0] == 'C' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(p.Cs) {
			return nil
		}
		return p.Cs[i]
	}
	return nil
}

func (p *proof) Exp(ret *big.Int, name string, exp, n *big.Int) bool {
	base := p.Base(name)
	if base == nil {
		return false
	}
	ret.Exp(base, exp, n)
	return true
}

func (p *proof) Names() []string {
	result := make([]string, 0, len(p.Cs))
	for i := range p.Cs {
		result = append(result, fmt.Sprintf("C%d", i))
	}

	return result
}
