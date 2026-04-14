export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const bookingId = searchParams.get('booking_id');

  if (!bookingId) {
    return Response.json({ error: 'booking_id is required' }, { status: 400 });
  }

  try {
    const response = await fetch(
      `http://localhost:8082/payments/snap-token?booking_id=${bookingId}`,
      {
        method: 'GET',
        cache: 'no-store',
      }
    );

    if (!response.ok) {
      return Response.json({ error: 'failed to fetch snap token' }, { status: 500 });
    }

    const payload = await response.json();
    const token = payload?.snap_token;

    if (!token) {
      return Response.json({ error: 'snap token not found' }, { status: 500 });
    }

    return Response.json({ token }, { status: 200 });
  } catch {
    return Response.json({ error: 'payment service unavailable' }, { status: 500 });
  }
}
