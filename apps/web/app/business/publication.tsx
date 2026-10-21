"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api, ApiError, Business, readable } from "../bookings/api";
import PublicProfileCard, { PublicProfile } from "./public-profile";
import { useDrafts } from "./drafts";
import { button, panel } from "./styles";
type Review = {
  published: boolean;
  profile: PublicProfile;
  review_token: string;
};

export default function Publication({ business }: { business: Business }) {
  const { saveAll } = useDrafts();
  const writable = ["BUSINESS_OWNER", "BUSINESS_ADMIN"].includes(business.role);
  const [confirmed, setConfirmed] = useState(false);
  const path = `/api/v1/businesses/${business.id}/publication`;
  const [review, setReview] = useState<Review | null>(null);
  const [busy, setBusy] = useState(true);
  const [message, setMessage] = useState("Loading saved profile for review...");
  const [version, setVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    api<Review>(path, { signal: controller.signal })
      .then((result) => {
        setReview(result.data);
        setMessage("");
      })
      .catch((error) => {
        if (!controller.signal.aborted) {
          setReview(null);
          setMessage(readable(error));
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
    return () => controller.abort();
  }, [path, version]);
  async function change(published: boolean) {
    if (published && (!review || !confirmed)) return;
    if (
      !published &&
      !window.confirm(
        "Unpublish this business profile? Its public link will stop showing the profile.",
      )
    )
      return;
    setBusy(true);
    try {
      const publish = async () => {
        let latest = review;
        if (published) {
          latest = (await api<Review>(path)).data;
          setReview(latest);
          const missing =
            latest.profile.services?.filter(
              (service) => !service.locations?.length,
            ) ?? [];
          if (missing.length)
            throw new Error(
              `Your edits were saved. Before publishing, choose a location for ${missing.map((service) => service.name).join(", ")} under Available at on its service card. Then publish again.`,
            );
        }
        const result = await api<Review>(path, {
          method: "PUT",
          body: JSON.stringify({
            published,
            ...(published ? { review_token: latest?.review_token } : {}),
          }),
        });
        setReview(result.data);
        setMessage(
          published
            ? "Reviewed profile published. You can share its public link."
            : "Profile unpublished. Its public link no longer shows your details.",
        );
      };
      if (published) await saveAll(publish);
      else await publish();
    } catch (error) {
      setMessage(
        error instanceof ApiError && error.status === 409
          ? "Saved details changed after this preview loaded. Refresh the preview, review it again, then publish."
          : error instanceof ApiError
            ? `${readable(error)} Refresh the preview to check the publication status.`
            : error instanceof Error
              ? error.message
              : "Could not save and publish. Try again.",
      );
    } finally {
      setConfirmed(false);
      setBusy(false);
    }
  }
  return (
    <section aria-label="Publishing review" className={panel}>
      <h2 className="text-2xl font-semibold">Review your business profile</h2>
      <p className="my-3 text-sm">
        Choose what customers see on your business page. Publishing makes your
        profile, services and locations visible to everyone.
      </p>
      <p className="mb-4 text-sm">
        Publish saves your pending workspace edits, refreshes this preview and
        publishes the updated profile. If a save fails, nothing new is
        published.
      </p>
      {message && (
        <p role="status" className="my-3 text-sm text-teal-900">
          {message}
        </p>
      )}
      <button
        disabled={busy}
        className={button}
        onClick={() => {
          setConfirmed(false);
          setReview(null);
          setBusy(true);
          setVersion((v) => v + 1);
        }}
      >
        Refresh publishing preview
      </button>
      {!review && !busy && writable && (
        <button className={button} onClick={() => void change(false)}>
          Unpublish profile
        </button>
      )}
      {review && (
        <div className="mt-5">
          <p className="my-4 font-semibold">
            Status: {review.published ? "Published" : "Not published"}
          </p>
          <PublicProfileCard preview profile={review.profile} />
          <div className="mt-4 rounded-xl border p-4">
            <h3 className="text-lg font-semibold">
              Services included in this publication
            </h3>
            <p className="my-2 text-sm">
              Customers see each service, its duration and where it is offered.
              Connect a location for every service, then set opening hours and
              booking rules in Scheduling.
            </p>
            {review.profile.services?.length ? (
              <ul className="divide-y">
                {review.profile.services.map((item) => (
                  <li className="break-words py-3" key={item.id}>
                    {item.name} -{" "}
                    {item.all_day ? "Whole day" : item.booking_options?.enabled
                      ? `Customer-selected length: ${item.booking_options.minimum_minutes}-${item.booking_options.maximum_minutes} minutes`
                      : `${item.duration_minutes} minutes`}
                    {!item.locations?.length && (
                      <p className="mt-2 font-medium text-amber-900">
                        Choose a branch under Available at on this service before publishing.
                      </p>
                    )}
                    <ul className="mt-2 pl-4 text-sm">
                      {item.locations?.map((location) => (
                        <li key={location.id}>
                          {location.name} ({location.timezone})
                        </li>
                      ))}
                    </ul>
                  </li>
                ))}
              </ul>
            ) : (
              <p>No active services to publish.</p>
            )}
          </div>
          {writable ? (
            <div className="mt-5 space-y-4">
              <label className="flex items-start gap-3">
                <input
                  type="checkbox"
                  className="mt-1 size-5 shrink-0"
                  disabled={busy}
                  checked={confirmed}
                  onChange={(event) => setConfirmed(event.target.checked)}
                />
                <span>
                  I want to save my pending edits and publish the updated
                  business profile.
                </span>
              </label>
              <div className="flex flex-wrap gap-3">
                <button
                  className={button}
                  disabled={busy || !confirmed}
                  onClick={() => void change(true)}
                >
                  {review.published
                    ? "Publish reviewed update"
                    : "Publish reviewed profile"}
                </button>
                {review.published && (
                  <button
                    className={button}
                    disabled={busy}
                    onClick={() => void change(false)}
                  >
                    Unpublish profile
                  </button>
                )}
              </div>
            </div>
          ) : (
            <p className="mt-4 text-sm">
              Only an owner or admin can publish or unpublish.
            </p>
          )}
          {review.published && (
            <p className="mt-5 break-all">
              <Link
                href={`/businesses/${business.id}`}
                target="_blank"
                rel="noopener noreferrer"
                className="text-teal-800 underline"
              >
                View public business profile
              </Link>
            </p>
          )}
        </div>
      )}
    </section>
  );
}
