export interface Trip {
  id: string;
  title: string;
  destination: string;
  price: number;
  date: string;
  available_slots: number;
}

export interface Booking {
  id: string;
  user_id: string;
  resource_id: string;
  total_amount: number;
  status: string;
  expires_at?: string;
  created_at: string;
  updated_at?: string;
}
