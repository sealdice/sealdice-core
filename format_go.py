import os
import subprocess


def scan_files(path):
    gofiles = []
    for root, dirs, files in os.walk(path):
        for file in files:
            if file.endswith(".go"):
                filepath = os.path.join(root, file)
                gofiles.append(filepath)
    return gofiles


def run_gofumpt(filename):
    try:
        formatted = subprocess.check_output(
            ["gofumpt", filename], stderr=subprocess.STDOUT
        )
        with open(filename, "wb") as file:
            file.write(formatted)
    except subprocess.CalledProcessError as e:
        print(f"Subprocess Error: {e.output.decode()}")
        exit(1)


if __name__ == "__main__":
    gofiles = scan_files(os.getcwd())
    total = len(gofiles)
    for index, file in enumerate(gofiles):
        print(f"Formatting \x1b[33m{file}\x1b[0m... ({index + 1}/{total})")
        run_gofumpt(file)
    print(f"All {total} files are formatted!")
