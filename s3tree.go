package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/a8m/tree"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"os"
)

var (
	// o : Output to file
	o = flag.String("o", "", "")
	// a : All files
	a = flag.Bool("a", false, "")
	// d : Dirs only
	d = flag.Bool("d", false, "")
	// f : Full path
	f = flag.Bool("f", false, "")
	// s : Show byte size
	s = flag.Bool("s", false, "")
	// h : Show SI size
	h = flag.Bool("h", false, "")
	// Q : Quote filename
	Q = flag.Bool("Q", false, "")
	// D : Show last mod
	D = flag.Bool("D", false, "")
	// C : Colorize
	C = flag.Bool("C", false, "")
	// L : Deep level
	L = flag.Int("L", 0, "")
	// U : No sort
	U = flag.Bool("U", false, "")
	// v : Version sort
	v = flag.Bool("v", false, "")
	// t : Last modification sort
	t = flag.Bool("t", false, "")
	// r : Reverse sort
	r = flag.Bool("r", false, "")
	// P : Matching pattern
	P = flag.String("P", "", "")
	// I : Ignoring pattern
	I = flag.String("I", "", "")
	// ignorecase : Ignore case-senstive
	ignorecase = flag.Bool("ignore-case", false, "")
	// dirsfirst : Dirs first sort
	dirsfirst = flag.Bool("dirsfirst", false, "")
	// sort : Sort by name or size
	sort = flag.String("sort", "", "")
	// S3 args
	bucket = flag.String("b", "", "")
	prefix = flag.String("p", "", "")
	region = flag.String("region", "us-east-1", "")
	profile = flag.String("profile", "tokenauth", "")
)

var usage = `Usage: s3tree -b bucket-name -p prefix(optional) -region region -profile IAM_Profile(optional) [options...]

Options:
    --------- S3 options ----------
    -b		    s3 bucket(required).
    -p		    s3 prefix.
    --region name   aws region(default to us-east-1).
		--profile
    ------- Listing options -------
    -a		    All files are listed.
    -d		    List directories only.
    -f		    Print the full path prefix for each file.
    -L		    Descend only level directories deep.
    -P		    List only those files that match the pattern given.
    -I		    Do not list files that match the given pattern.
    --ignore-case   Ignore case when pattern matching.
    -o filename	    Output to file instead of stdout.
    -------- File options ---------
    -Q		    Quote filenames with double quotes.
    -s		    Print the size in bytes of each file.
    -h		    Print the size in a more human readable way.
    -D		    Print the date of last modification or (-c) status change.
    ------- Sorting options -------
    -v		    Sort files alphanumerically by version.
    -t		    Sort files by last modification time.
    -U		    Leave files unsorted.
    -r		    Reverse the order of the sort.
    --dirsfirst	    List directories before files (-U disables).
    --sort X	    Select sort: name,size,version.
    ------- Graphics options ------
    -i		    Don't print indentation lines.
    -C		    Turn colorization on always.
`

func main() {
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()
	if len(*bucket) == 0 {
		err := errors.New("-b(s3 bucket) is required")
		errAndExit(err)
	}

	var noPrefix = len(*prefix) == 0
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config:  aws.Config{Region: region},
		Profile: *profile,
	}))
	svc := s3.New(sess)
	spin := NewSpin()
	resp, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket: bucket,
		Prefix: prefix,
	})
	spin.Done()
	var fs = NewFs()
	if err != nil {
		errAndExit(err)
	} else {
		// Loop over s3 object
		for _, obj := range resp.Contents {
			key := *obj.Key
			if noPrefix {
				key = fmt.Sprintf("%s/%s", *bucket, key)
			}
			fs.addFile(key, obj)
		}
	}
	var nd, nf int
	rootDir := *prefix
	if noPrefix {
		rootDir = *bucket
	}
	// Output file
	var outFile = os.Stdout
	if *o != "" {
		outFile, err = os.Create(*o)
		if err != nil {
			errAndExit(err)
		}
	}
	defer outFile.Close()
	opts := &tree.Options{
		Fs:        fs,
		OutFile:   outFile,
		All:       *a,
		DirsOnly:  *d,
		FullPath:  *f,
		ByteSize:  *s,
		UnitSize:  *h,
		Quotes:    *Q,
		LastMod:   *D,
		Colorize:  *C,
		DeepLevel: *L,
		NoSort:    *U,
		ReverSort: *r,
		DirSort:   *dirsfirst,
		VerSort:   *v || *sort == "version",
		NameSort:  *sort == "name",
		SizeSort:  *sort == "size",
		ModSort:   *t,
		Pattern:   *P,
		IPattern:  *I,
	}
	inf := tree.New(rootDir)
	if d, f := inf.Visit(opts); f != 0 {
		nd, nf = nd+d-1, nf+f
	}
	inf.Print(opts)
	// print footer
	footer := fmt.Sprintf("\n%d directories", nd)
	if !opts.DirsOnly {
		footer += fmt.Sprintf(", %d files", nf)
	}
	fmt.Fprintf(outFile, footer)
}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func errAndExit(err error) {
	fmt.Fprintf(os.Stderr, "s3tree: \"%s\"\n", err)
	os.Exit(1)
}
