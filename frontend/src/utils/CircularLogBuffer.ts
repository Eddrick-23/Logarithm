// CircularLogBuffer is created to store logs in a circular queue and to minimise
// the creation of new arrays when new logs comes in
export class CircularLogBuffer<T> {
    private buffer: T[];
    private head: number = 0;
    private size: number = 0;
    private maxSize: number;

    constructor(maxSize: number) {
        this.maxSize = maxSize;
        this.buffer = new Array(maxSize);
    }

    add(item: T) {
        this.buffer[this.head] = item;
        this.head = (this.head + 1) % this.maxSize;
        this.size = Math.min(this.size + 1, this.maxSize);
    }

    toArrayNewestFirst(): T[] {
        const result = new Array(this.size);
        let currentIndex = this.head - 1;

        for (let i = 0; i < this.size; i++) {
            if (currentIndex < 0) {
                currentIndex = this.maxSize - 1;
            }
            result[i] = this.buffer[currentIndex];
            currentIndex--;
        }

        // sort the logs based on timestamp in descending order since it is not guaranteed that incoming logs timestamps are in chronological order
        return result.sort((a, b) => b.timestamp - a.timestamp);
    }
}
