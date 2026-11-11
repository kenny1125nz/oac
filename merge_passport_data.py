import json, glob, os, sys, argparse

def main():
    parser = argparse.ArgumentParser(description="Merge passport photo requirement JSON files")
    parser.add_argument("--input-dir", default="/tmp", help="Directory containing passport_*.json files")
    parser.add_argument("--output", default="/tmp/passport_requirements.json", help="Output file path")
    args = parser.parse_args()

    pattern = os.path.join(args.input_dir, "passport_*.json")
    files = sorted(glob.glob(pattern))
    files = [f for f in files if os.path.basename(f) != os.path.basename(args.output)]

    items = []
    for f in files:
        try:
            with open(f) as fh:
                items.append(json.load(fh))
            os.remove(f)
        except Exception as e:
            print(f"skip {f}: {e}", file=sys.stderr)

    with open(args.output, "w") as f:
        json.dump(items, f, indent=2, ensure_ascii=False)

    print(json.dumps(items, indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
