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
    float elapsed;
    int beats;

    void Awake()
    {
        string value = Config.Bind("General", "Value", "default", "Written by the matrix regress.").Value;
        Logger.LogInfo("matrix " + Variant.Name + " cfg=" + value);
        // A data file shipped in a folder of its own, which r2modman's layout puts beside the plugin.
        string beside = Path.Combine(Path.GetDirectoryName(Info.Location), "matrix-data.txt");
        Logger.LogInfo("matrix " + Variant.Name + " beside=" + File.Exists(beside));
#if SAVE
        // The game's own save format: Easy Save 3 with the game's settings, the way Lethal Company writes its slots.
        ES3.Save("GroupCredits", 1234, "LCSaveFile1");
        ES3.Save("FileGameVers", 72, "LCSaveFile1");
        Logger.LogInfo("matrix " + Variant.Name + " wrote LCSaveFile1");
#endif
#if THROW
        throw new System.InvalidOperationException("matrix probe failed in Awake on purpose");
#endif
    }

#if BEAT || QUIT
    void Update()
    {
        elapsed += Time.unscaledDeltaTime;
        if (elapsed < 1f)
        {
            return;
        }
        elapsed = 0;
        beats++;
#if BEAT
        Logger.LogWarning("matrix heartbeat " + beats);
        if (beats == 3)
        {
            Logger.LogError("matrix error line");
        }
#endif
#if QUIT
        if (beats == 8)
        {
            Logger.LogInfo("matrix quitting");
            Application.Quit();
        }
#endif
    }
#endif
}
