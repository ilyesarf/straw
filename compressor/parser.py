
def parse_stream(stream_file="../stream.txt"):
    stream = [ev.strip() for ev in open(stream_file, 'r').readlines()]
    return stream


if __name__ == "__main__":
    raw_snapshot = parse_stream()
    print(raw_snapshot[:2])