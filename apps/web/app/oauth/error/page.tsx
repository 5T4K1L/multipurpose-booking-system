import Link from "next/link";

export default function OAuthErrorPage() {
  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-10">
      <section className="w-full max-w-md">
        <h1 className="text-3xl font-semibold">
          Sign-in could not be completed
        </h1>
        <p className="mt-4">
          Please return to sign in and try again. If you already use a password
          for this account, sign in with that password. Accounts are not
          automatically linked by email.
        </p>
        <Link
          href="/login"
          className="mt-6 inline-block underline underline-offset-4"
        >
          Return to sign in
        </Link>
      </section>
    </main>
  );
}
