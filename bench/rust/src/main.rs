// Rust's u128 and i128 on the inputs of fat's BenchmarkRust* cases.
// The checked cases panic on overflow as fat does, the wrapping ones wrap
// as fat's u forms, / and % panic on zero and on MIN / -1. Every case is
// checked against fat's results before it is timed, and prints lines in
// the format of go test so benchstat -col /impl puts both sides together.
// Run by ../run.sh: fat-rust-bench <dump dir> <count> <milliseconds>
use std::hint::black_box;
use std::time::{Duration, Instant};

// m indexes the 2^18 inputs of every case, as in the Go benchmarks
const M: usize = (1 << 18) - 1;
// AGREE is how many of fat's results the dump holds, agreeN in Go
const AGREE: usize = 1 << 14;

struct Ctx {
    dir: String,
    procs: String,
    count: usize,
    target: Duration,
}

// load reads x, y and fat's results of one case
fn load(ctx: &Ctx, name: &str) -> (Vec<u128>, Vec<u128>, Vec<u128>) {
    let path = format!("{}/{}.bin", ctx.dir, name.replace('/', "_"));
    let b = std::fs::read(&path).unwrap_or_else(|e| panic!("{path}: {e}"));
    let v: Vec<u128> = b
        .chunks_exact(16)
        .map(|c| u128::from_le_bytes(c.try_into().unwrap()))
        .collect();
    let n = M + 1;
    assert_eq!(v.len(), 2 * n + AGREE, "{path}: wrong size");
    (v[..n].to_vec(), v[n..2 * n].to_vec(), v[2 * n..].to_vec())
}

// run is the timed loop, each result kept by black_box as Go keeps it in a sink
#[inline(never)]
fn run<T: Copy, R, F: Fn(T, T) -> R>(n: usize, x: &[T], y: &[T], f: &F) {
    for i in 0..n {
        black_box(f(x[i & M], y[i & M]));
    }
}

// bench grows n as go test does until a run takes the target time, count times
fn bench<T: Copy, R, F: Fn(T, T) -> R>(ctx: &Ctx, name: &str, x: &[T], y: &[T], f: F) {
    let (x, y) = (black_box(x), black_box(y));
    for _ in 0..ctx.count {
        let mut n = 1usize;
        let mut el;
        loop {
            let t = Instant::now();
            run(n, x, y, &f);
            el = t.elapsed();
            if el >= ctx.target || n >= 2_000_000_000 {
                break;
            }
            let per = el.as_nanos().max(1) as f64 / n as f64;
            let want = (ctx.target.as_nanos() as f64 * 1.2 / per) as usize;
            n = want.clamp(n + 1, n * 100);
        }
        let ns = el.as_nanos() as f64 / n as f64;
        println!(
            "Benchmark{name}/impl=rust-{}\t{n}\t{ns:.4} ns/op",
            ctx.procs
        );
    }
}

fn unsigned<F: Fn(u128, u128) -> u128>(ctx: &Ctx, name: &str, f: F) {
    let (x, y, want) = load(ctx, name);
    for i in 0..AGREE {
        assert_eq!(f(x[i], y[i]), want[i], "{name}: value {i} differs from fat");
    }
    bench(ctx, name, &x, &y, f);
}

fn signed<F: Fn(i128, i128) -> i128>(ctx: &Ctx, name: &str, f: F) {
    let (x, y, want) = load(ctx, name);
    let x: Vec<i128> = x.iter().map(|&v| v as i128).collect();
    let y: Vec<i128> = y.iter().map(|&v| v as i128).collect();
    for i in 0..AGREE {
        assert_eq!(
            f(x[i], y[i]) as u128,
            want[i],
            "{name}: value {i} differs from fat"
        );
    }
    bench(ctx, name, &x, &y, f);
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    if args.len() != 4 {
        eprintln!("usage: fat-rust-bench <dump dir> <count> <milliseconds>");
        std::process::exit(2);
    }
    let dir = args[1].clone();
    let procs = std::fs::read_to_string(format!("{dir}/procs"))
        .expect("procs")
        .trim()
        .to_string();
    let ctx = Ctx {
        dir,
        procs,
        count: args[2].parse().expect("count"),
        target: Duration::from_millis(args[3].parse().expect("milliseconds")),
    };

    let u = "RustUint128/";
    unsigned(&ctx, &format!("{u}op=Add/in=fit"), |a, b| {
        a.checked_add(b).unwrap()
    });
    unsigned(&ctx, &format!("{u}op=Sub/in=fit"), |a, b| {
        a.checked_sub(b).unwrap()
    });
    unsigned(&ctx, &format!("{u}op=Mul/in=fit"), |a, b| {
        a.checked_mul(b).unwrap()
    });
    unsigned(&ctx, &format!("{u}op=Addu/in=mixed"), |a, b| {
        a.wrapping_add(b)
    });
    unsigned(&ctx, &format!("{u}op=Subu/in=mixed"), |a, b| {
        a.wrapping_sub(b)
    });
    unsigned(&ctx, &format!("{u}op=Mulu/in=mixed"), |a, b| {
        a.wrapping_mul(b)
    });
    for set in ["mixed", "div64", "full"] {
        unsigned(&ctx, &format!("{u}op=Div/in={set}"), |a, b| a / b);
        unsigned(&ctx, &format!("{u}op=Mod/in={set}"), |a, b| a % b);
    }

    let s = "RustInt128/";
    signed(&ctx, &format!("{s}op=Add/in=fit"), |a, b| {
        a.checked_add(b).unwrap()
    });
    signed(&ctx, &format!("{s}op=Sub/in=fit"), |a, b| {
        a.checked_sub(b).unwrap()
    });
    signed(&ctx, &format!("{s}op=Mul/in=fit"), |a, b| {
        a.checked_mul(b).unwrap()
    });
    signed(&ctx, &format!("{s}op=Addu/in=mixed"), |a, b| {
        a.wrapping_add(b)
    });
    signed(&ctx, &format!("{s}op=Subu/in=mixed"), |a, b| {
        a.wrapping_sub(b)
    });
    signed(&ctx, &format!("{s}op=Mulu/in=mixed"), |a, b| {
        a.wrapping_mul(b)
    });
    for set in ["mixed", "div64"] {
        signed(&ctx, &format!("{s}op=Div/in={set}"), |a, b| a / b);
        signed(&ctx, &format!("{s}op=Mod/in={set}"), |a, b| a % b);
    }
}
