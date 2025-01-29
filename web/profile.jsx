import { useEffect, useState } from "react";

export default function Profile() {
    const [user, setUser] = useState(null);
    const [pastes, setPastes] = useState([]);

    useEffect(() => {
        fetch("/profile", { credentials: "include" })
            .then((res) => res.json())
            .then((data) => {
                setUser({ name: data.name, email: data.email });
                setPastes(data.pastes);
            })
            .catch((err) => console.error("Error fetching profile:", err));
    }, []);

    const handleDelete = (id) => {
        fetch(`/pastes/${id}/delete`, {
            method: "POST",
            credentials: "include",
        })
            .then((res) => {
                if (res.ok) {
                    setPastes(pastes.filter((paste) => paste._id !== id));
                }
            })
            .catch((err) => console.error("Error deleting paste:", err));
    };

    return (
        <div className="min-h-screen bg-gray-900 text-white p-6">
            <div className="max-w-4xl mx-auto">
                <h1 className="text-3xl font-bold mb-4">Profile</h1>
                {user ? (
                    <div className="bg-gray-800 p-6 rounded-lg shadow-lg">
                        <p className="text-lg mb-2">Name: {user.name}</p>
                        <p className="text-lg mb-4">Email: {user.email}</p>
                        <h2 className="text-2xl font-semibold mb-3">My pastes:</h2>
                        {pastes.length > 0 ? (
                            <div className="space-y-4">
                                {pastes.map((paste) => (
                                    <div key={paste._id} className="bg-black p-4 rounded-lg">
                                        <h3 className="text-xl font-semibold">{paste.title}</h3>
                                        <p className="text-gray-400">{paste.content}</p>
                                        <p className="text-sm text-gray-500 mt-2">
                                            Created at: {new Date(paste.createdAt).toLocaleString()}
                                        </p>
                                        <div className="flex space-x-2 mt-3">
                                            <a
                                                href={`/pastes/${paste._id}/edit`}
                                                className="bg-blue-500 hover:bg-blue-700 text-white px-3 py-1 rounded"
                                            >
                                                Edit
                                            </a>
                                            <button
                                                onClick={() => handleDelete(paste._id)}
                                                className="bg-red-500 hover:bg-red-700 text-white px-3 py-1 rounded"
                                            >
                                                Delete
                                            </button>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <p className="text-gray-500">No pastes found.</p>
                        )}
                    </div>
                ) : (
                    <p>Loading...</p>
                )}
            </div>
        </div>
    );
}
