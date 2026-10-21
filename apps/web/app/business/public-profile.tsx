import Image from "next/image";
export type PublicLocation = { id: string; name: string; timezone: string };
export type PublicService = {
 all_day?:boolean;
  booking_options?: {
    enabled: boolean;
    minimum_minutes: number;
    maximum_minutes: number;
  };
  id: string;
  name: string;
  duration_minutes: number;
  locations: PublicLocation[];
};
export type PublicProfile = {
  image_version?:string;
  id: string;
  name: string;
  timezone: string;
  tagline: string;
  description: string;
  contact_email: string;
  contact_phone: string;
  website: string;
  services: PublicService[];
};

// Shared text-only presentation for reviewed and published snapshots.
export default function PublicProfileCard({
  profile,
  preview=false,
}: {
  profile: PublicProfile;
  preview?:boolean;
}) {
  return (
    <article
      aria-label="Business profile preview"
      className="min-w-0 break-words rounded-2xl border border-slate-200 bg-white p-5 sm:p-8"
    >
      {profile.image_version&&<Image unoptimized src={`${process.env.NEXT_PUBLIC_API_BASE_URL??"http://localhost:8080"}/api/v1/${preview?"businesses":"public/businesses"}/${profile.id}/image?v=${profile.image_version}`} alt={`${profile.name} business photo`} width={1200} height={600} className="mb-6 max-h-96 w-full rounded-xl object-cover"/>}
      <h3 className="text-3xl font-semibold">{profile.name}</h3>
      <p className="mt-2 text-sm text-slate-600">
        Timezone: {profile.timezone}
      </p>
      {profile.tagline && (
        <p className="mt-5 text-xl font-medium">{profile.tagline}</p>
      )}
      {profile.description && (
        <p className="mt-4 whitespace-pre-wrap">{profile.description}</p>
      )}
      <dl className="mt-6 grid min-w-0 gap-4 sm:grid-cols-2">
        {(
          [
            ["Contact email", profile.contact_email],
            ["Contact phone", profile.contact_phone],
            ["Website", profile.website],
          ] as const
        ).filter(([,value])=>!!value).map(([label, value]) => (
          <div key={label} className="min-w-0">
            <dt className="text-sm font-semibold text-slate-600">{label}</dt>
            <dd className="mt-1 [overflow-wrap:anywhere]">
              {value || "Not provided"}
            </dd>
          </div>
        ))}
      </dl>
    </article>
  );
}
