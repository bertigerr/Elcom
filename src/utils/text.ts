export function normalizeHeader(input: string): string {
  return input
    .toUpperCase()
    .replace(/Ё/g, 'Е')
    .replace(/[×хХ*]/g, 'X')
    .replace(/ММ²|MM²|КВ\.\s*ММ|КВММ|MM2/g, 'MM2')
    .replace(/["'`«»]/g, ' ')
    .replace(/[^A-ZА-Я0-9X\-/\s.]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

export function normalizeCode(input: string): string {
  return input
    .toUpperCase()
    .replace(/\s+/g, '')
    .replace(/[×хХ*]/g, 'X')
    .replace(/[^A-ZА-Я0-9\-_/]/g, '');
}

export function tokenize(input: string): string[] {
  return normalizeHeader(input)
    .split(' ')
    .map((s) => s.trim())
    .filter((s) => s.length >= 2);
}

export function diceCoefficient(a: string, b: string): number {
  if (!a || !b) {
    return 0;
  }
  if (a === b) {
    return 1;
  }

  const pairs = (s: string): string[] => {
    const p: string[] = [];
    for (let i = 0; i < s.length - 1; i += 1) {
      p.push(s.slice(i, i + 2));
    }
    return p;
  };

  const aPairs = pairs(a);
  const bPairs = pairs(b);
  if (!aPairs.length || !bPairs.length) {
    return 0;
  }

  const bMap = new Map<string, number>();
  for (const p of bPairs) {
    bMap.set(p, (bMap.get(p) ?? 0) + 1);
  }

  let intersection = 0;
  for (const p of aPairs) {
    const count = bMap.get(p) ?? 0;
    if (count > 0) {
      intersection += 1;
      bMap.set(p, count - 1);
    }
  }

  return (2 * intersection) / (aPairs.length + bPairs.length);
}

export function looksLikeCode(input: string): boolean {
  const trimmed = input.trim();
  return /[A-Za-z]/.test(trimmed) && /\d/.test(trimmed) && /^[A-Za-zА-Яа-я0-9\-_/\.\s]{3,}$/.test(trimmed);
}
