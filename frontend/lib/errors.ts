/** Mengambil pesan galat dari respons server ({"error": "..."}), atau null bila tidak ada. */
export function serverMessage(error: unknown): string | null {
  const data = (error as { response?: { data?: { error?: unknown } } })?.response?.data;
  return typeof data?.error === 'string' && data.error ? data.error : null;
}

export function statusOf(error: unknown): number | undefined {
  return (error as { response?: { status?: number } })?.response?.status;
}
