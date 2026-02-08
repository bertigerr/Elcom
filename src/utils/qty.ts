const UNIT_PATTERN = /(шт|штук|pcs|pc|м\.?|метр|kg|кг|уп\.?|компл\.?)/i;

export interface ParsedQty {
  qty: number | null;
  unit: string | null;
  qtyRaw: string | null;
}

export function parseQty(input: string): ParsedQty {
  const line = input.replace(/\u00A0/g, ' ');
  const numberPattern = '(\\d{1,3}(?:[\\s.,]\\d{3})+|\\d+(?:[.,]\\d+)?)';
  const withUnitMatches = Array.from(
    line.matchAll(new RegExp(`(?<![A-Za-zА-Яа-я0-9.,])${numberPattern}\\s*${UNIT_PATTERN.source}`, 'gi')),
  );
  const numericMatches = Array.from(
    line.matchAll(new RegExp(`(?<![A-Za-zА-Яа-я0-9.,])${numberPattern}(?![A-Za-zА-Яа-я0-9.,])`, 'g')),
  );
  const qtyMatch = withUnitMatches.length
    ? withUnitMatches[withUnitMatches.length - 1]
    : numericMatches.length
      ? numericMatches[numericMatches.length - 1]
      : null;

  let qty: number | null = null;
  let qtyRaw: string | null = null;

  if (qtyMatch) {
    qtyRaw = qtyMatch[0]?.trim() ?? null;
    const primary = (qtyMatch[1] ?? qtyRaw ?? '').trim();
    const normalized = normalizeNumericToken(primary);
    const parsed = Number(normalized);
    qty = Number.isFinite(parsed) ? parsed : null;
  }

  const unitMatch = line.match(UNIT_PATTERN);
  const unit = unitMatch ? normalizeUnit(unitMatch[1]) : null;

  return { qty, unit, qtyRaw };
}

export function normalizeUnit(unit: string): string {
  const u = unit.toLowerCase();
  if (['шт', 'штук', 'pcs', 'pc'].includes(u)) {
    return 'шт';
  }
  if (['м', 'м.', 'метр'].includes(u)) {
    return 'м';
  }
  if (['kg', 'кг'].includes(u)) {
    return 'кг';
  }
  if (['уп', 'уп.'].includes(u)) {
    return 'уп';
  }
  return u;
}

function normalizeNumericToken(token: string): string {
  const compact = token.replace(/\s+/g, '');

  if (/^\d{1,3}(?:\.\d{3})+$/.test(compact)) {
    return compact.replace(/\./g, '');
  }

  if (/^\d{1,3}(?:,\d{3})+$/.test(compact)) {
    return compact.replace(/,/g, '');
  }

  if (compact.includes(',') && !compact.includes('.')) {
    return compact.replace(',', '.');
  }

  return compact;
}
