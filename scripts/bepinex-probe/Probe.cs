// A BepInEx plugin the Lethal Company matrix regress (scripts/regress-bepinex.sh) builds once per variant. Variant.cs,
// written by the script, names the variant; the defines pick what it does. Every variant logs its config value, so a
// run shows where BepInEx found the plugin's .cfg and what Mortar's editor wrote into it.
using System.IO;
using BepInEx;
using UnityEngine;

namespace MortarMatrix;

[BepInPlugin("tech.rethunk.mortar.matrix." + Variant.Name, "Matrix " + Variant.Name, "1.0.0")]
public class MatrixProbe : BaseUnityPlugin
{
#if BEAT
    float elapsed;
    int beats;
#endif

    void Awake()
    {
        string value = Config.Bind("General", "Value", "default", "Written by the matrix regress.").Value;
        Logger.LogInfo("matrix " + Variant.Name + " cfg=" + value);
        float ratio = Config.Bind("General", "Ratio", 0.5f, "A float, for Mortar's typed editor.").Value;
        Logger.LogInfo("matrix " + Variant.Name + " ratio=" + ratio.ToString(System.Globalization.CultureInfo.InvariantCulture));
        // A data file shipped in a folder of its own, which r2modman's layout puts beside the plugin.
        string beside = Path.Combine(Path.GetDirectoryName(Info.Location), "matrix-data.txt");
        Logger.LogInfo("matrix " + Variant.Name + " beside=" + File.Exists(beside));
#if SAVE
        // The game's own save format: Easy Save 3 with the game's settings, the way Lethal Company writes its slots.
        ES3.Save("GroupCredits", 1234, "LCSaveFile1");
        ES3.Save("FileGameVers", 72, "LCSaveFile1");
        Logger.LogInfo("matrix " + Variant.Name + " wrote LCSaveFile1");
#endif
#if QUIT
        // With HideManagerGameObject=false, BepInEx loses its manager object, and every plugin on it, when Lethal Company
        // loads its first scene, so the quit timer runs on an object of its own that scene loads leave alone.
        var clock = new GameObject("MatrixQuit") { hideFlags = HideFlags.HideAndDontSave };
        DontDestroyOnLoad(clock);
        clock.AddComponent<MatrixQuit>().Log = Logger;
#endif
#if THROW
        throw new System.InvalidOperationException("matrix probe failed in Awake on purpose");
#endif
    }

#if BEAT
    void Update()
    {
        elapsed += Time.unscaledDeltaTime;
        if (elapsed < 1f)
        {
            return;
        }
        elapsed = 0;
        beats++;
        Logger.LogWarning("matrix heartbeat " + beats);
        if (beats == 3)
        {
            Logger.LogError("matrix error line");
        }
    }
#endif
}

#if QUIT
public class MatrixQuit : MonoBehaviour
{
    public BepInEx.Logging.ManualLogSource Log;
    float elapsed;
    int beats;

    // The clock starts at the main menu, so the regress can query the bridge there before the game quits.
    void Update()
    {
        if (UnityEngine.SceneManagement.SceneManager.GetActiveScene().name != "MainMenu")
        {
            return;
        }
        elapsed += Time.unscaledDeltaTime;
        if (elapsed < 1f)
        {
            return;
        }
        elapsed = 0;
        if (++beats == 8)
        {
            Log.LogInfo("matrix quitting");
            Application.Quit();
        }
    }
}
#endif
