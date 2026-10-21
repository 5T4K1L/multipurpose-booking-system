"use client";
import { useParams } from "next/navigation";
import CustomerBookings from "../workspace";
export default function Page() {
  const { id } = useParams<{ id: string }>();
  return <CustomerBookings key={id} id={id} />;
}
